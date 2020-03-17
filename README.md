# ProjectSpicyDNS
Golang code meant to keep DNS caches hot, to not resort to calling back to the DNS root tree


## nitty-gritty details
In theory, I would use ANY, requests, but cloudflare, in their infinite wisdom, declared it as deprecated and I have to now assume it is not useable, even if cloudflare is not the lone authority on network infrastructure. (reference: http://imagen.click/d/cloudflareanydeprecation)

With this in mind, I am forced to pick and choose which queries are important to me. A and AAAA records are obvious things to keep, but I also would like to have CNAME, TXT and MX. NS should be aquired from any of the previous queries and stored alongside the rest of the data. This will eventually be in a config file with the ability to pick-and-choose different queries.