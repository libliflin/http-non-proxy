# http-non-proxy
An explicit http proxy for when you want to mitm yourself.

`go get github.com/libliflin/http-non-proxy`

`http-non-proxy -f localhost:12012 -t https://www.google.com`

I use this because dealing with java cacerts is a huge pain with old application servers. I'd rather go handle my ssl through my OS. 

Links:
* [go hands off ssl to your OS](https://github.com/golang/go/blob/master/src/crypto/x509/root_windows.go)
* [if you have to use cert locations in java, use a gui](http://portecle.sourceforge.net/)
* [localhost is good enough security](http://security.stackexchange.com/questions/58261/is-it-secure-to-use-no-authentication-for-services-listening-only-on-localhost)
* [certicom ssl doesn't support SHA-256](https://community.oracle.com/thread/3673769?start=0&tstart=0)
