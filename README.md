# mitm
An explicit http proxy for when you want to mitm yourself.

`go get github.com/libliflin/mitm/mitm`

`mitm -f localhost:12012 -t https://www.google.com`

I use this because dealing with java cacerts is a huge pain with old application servers. I'd rather go handle my ssl through my OS. 

## Now featuring streaming!

to test streaming:

    cd %GOPATH%\src\github.com\libliflin\mitm\streamtest
    go run servertest.go

You should see output like:

    2016/02/08 22:14:46 Testing mitm streaming with locks.
    2016/02/08 22:14:46 DOWNLOAD: lock initialized as locked
    2016/02/08 22:14:46 UPLOAD:   lock initialized as locked
    2016/02/08 22:14:46 UPLOAD:   waiting on client after first file upload chunk sent until it is verified on server.
    2016/02/08 22:14:46 UPLOAD:   first upload chunk verified, unlocking upload lock.
    2016/02/08 22:14:46 UPLOAD:   lock unlocked on client, sending second chunk.
    2016/02/08 22:14:46 UPLOAD:   upload stream check completed.
    2016/02/08 22:14:46 DOWNLOAD: locking on server after flushing first greeting.
    2016/02/08 22:14:46 DOWNLOAD: unlocking on client after validating first greeting.
    2016/02/08 22:14:46 DOWNLOAD: lock unlocked on server, sending second greeting.
    2016/02/08 22:14:46 DOWNLOAD: verification of second greeting complete.
    2016/02/08 22:14:46 All tests passed.

Links:
* [go hands off ssl to your OS](https://github.com/golang/go/blob/master/src/crypto/x509/root_windows.go)
* [if you have to use cert locations in java, use a gui](http://portecle.sourceforge.net/)
* [localhost is good enough security](http://security.stackexchange.com/questions/58261/is-it-secure-to-use-no-authentication-for-services-listening-only-on-localhost)
* [certicom ssl doesn't support SHA-256](https://community.oracle.com/thread/3673769?start=0&tstart=0)
