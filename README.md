# ProjectSpicyDNS
Golang code meant to keep DNS caches hot, to not resort to calling back to the DNS root tree


## nitty-gritty details
In theory, I would use ANY, requests, but cloudflare, in their infinite wisdom, declared it as deprecated and I have to now assume it is not useable, even if cloudflare is not the lone authority on network infrastructure. (reference: https://imagen.click/d/cloudflareanydeprecation)

With this in mind, I am forced to pick and choose which queries are important to me. A and AAAA records are obvious things to keep, but I also would like to have CNAME queries. NS should be aquired from any of the previous queries and stored alongside the rest of the data, and MX/TXT queries are not considered time sensitive. This will eventually be in a config file with the ability to pick-and-choose different queries.

In theory, in a low-memory environment, just caching NS records would significantly speed up queries while using less memory, but I am optimizing this for total caching capacity, bar none.

It could be argued that the number of domains that this would actually enhance is very small, but in a world of 5 minute TTL's and COVID-19 wiping out schools for a month, I ask you, why not do it *anyway*...

Credit to Miek Gieben for his DNS library, as it is the backbone of this system.