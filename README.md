Installing from source
----------------------

To install, run

    $ go get github.com/zyong/miniproxygo

Build

    $ go install github.com/zyong/miniproxygo@latest

You will now find a `miniproxygo` binary in your `$GOPATH/bin` directory. At the meanwhile, you should add a conf file in your conf directory ,named proxy.conf, you can copy file from conf directory in the root.

Usage
-----

Start proxy

    $ miniproxy 

Run `miniproxy -help` for more information.
