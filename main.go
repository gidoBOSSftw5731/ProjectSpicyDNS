package main

import (
	"archive/zip"
	"bufio"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gidoBOSSftw5731/log"
	"github.com/jinzhu/configor"
	_ "github.com/lib/pq"
	"github.com/miekg/dns"
)

var config = struct {
	DB struct {
		User     string `default:"spicydns"`
		Password string `required:"true" env:"DBPassword" default:"Sp1cyDn5"`
		Port     string `default:"5432"`
		IP       string `default:"127.0.0.1"`
	}
	Threads        int    `default:"25"`
	Nameserver     string `default:"127.0.0.1"`
	NameserverPort string `default::"53"`
}{}

var (
	domainQueue chan string
	db          *sql.DB
	queries     = []uint16{dns.TypeA, dns.TypeAAAA, dns.TypeCNAME}
)

const (
	domListZip = "http://s3.amazonaws.com/alexa-static/top-1m.csv.zip"
)

func main() {
	configor.Load(&config, "config.yml")
	log.SetCallDepth(4)
	tmp := os.TempDir()
	zipPath := path.Join(tmp, "domList.zip")
	domainQueue = make(chan string)

	err := downloadFile(zipPath, domListZip)
	if err != nil {
		log.Fatalln("Couldn't download file! ", err)
	}

	csvArr, err := unzip(zipPath, tmp)

	if len(csvArr) != 1 {
		log.Fatalln("Error! multiple files extracted!")
	} else if err != nil {
		log.Fatalln("Error extracting! ", err)
	}

	// csv file downloaded and extracted

	var entries []string
	csvFile, _ := os.Open(csvArr[0])
	reader := csv.NewReader(bufio.NewReader(csvFile))

	// yes, I know it's single-threaded, but it's fast enough that I dont care
	for {
		line, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}
		entries = append(entries, line[1])
	}

	// filesystems are slow, so let's just do these in the background
	// they're good practice, not a necessity
	go os.Remove(zipPath)
	go os.Remove(csvArr[0])

	// now all the domains are in an array (slice?)

	db, err = mkDB()

	initQuery(entries)
}

func mkDB() (*sql.DB, error) {
	return sql.Open("pq", fmt.Sprintf("%s:%s@tcp(%s:%s)/spicydns",
		config.DB.User, config.DB.Password, config.DB.IP, config.DB.Port))
	/*
		CREATE TABLE records (
			domain text,
			recordtype int,
			record text,
			ttl int,
			isExpired bool
		);

	*/
}

// queryer is a function that queries all input queries from a channel.
// These are workers, they are as many of these as there are threads defined in the config
// queryer means one that queries (query-er)
func queryer(id int, jobs <-chan string) {
	//dnsConfig, _ := dns.ClientConfigFromFile(filepath.Join(os.TempDir(), "resolv.conf"))
	c := new(dns.Client)

	for domain := range jobs {
		m := new(dns.Msg)
		m.SetQuestion(domain+".", dns.TypeNS)
		m.RecursionDesired = true
		r, _, _ := c.Exchange(m, net.JoinHostPort(config.Nameserver, config.NameserverPort))
		for _, answer := range r.Answer {
			db.Exec("INSERT INTO records (domain, recordtype, record, ttl, isExpired) VALUES (?, ?, ?, ?, ?)",
			 answer.Header().Name,answer.Header().Rrtype, answer.)
		}
		for _, queryType := range queries {
			m := new(dns.Msg)
			m.SetQuestion(domain+".", queryType)
			m.RecursionDesired = true
			r, _, _ := c.Exchange(m, net.JoinHostPort(config.Nameserver, config.NameserverPort))

		}
	}
}

func initQuery(domains []string) {
	for w := 0; w < config.Threads; w++ {
		go queryer(w, domainQueue)
	}

	for _, domain := range domains {
		domainQueue <- domain
	}

}

// downloadFile will download a url to a local file. It's efficient because it will
// write as it downloads and not load the whole file into memory.
func downloadFile(filepath string, url string) error {

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	log.Traceln("Downloaded")
	return err
}

// unzip will decompress a zip archive, moving all files and folders
// within the zip file (parameter 1) to an output directory (parameter 2).
func unzip(src string, dest string) ([]string, error) {

	var filenames []string

	r, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}
	defer r.Close()

	for _, f := range r.File {

		// Store filename/path for returning and using later on
		fpath := filepath.Join(dest, f.Name)

		// Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return filenames, fmt.Errorf("%s: illegal file path", fpath)
		}

		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {
			// Make Folder
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		// Make File
		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return filenames, err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return filenames, err
		}

		rc, err := f.Open()
		if err != nil {
			return filenames, err
		}

		_, err = io.Copy(outFile, rc)

		// Close the file without defer to close before next iteration of loop
		outFile.Close()
		rc.Close()

		if err != nil {
			return filenames, err
		}
	}
	return filenames, nil
}
