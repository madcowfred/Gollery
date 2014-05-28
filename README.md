Gollery
=======

Gollery is a (somewhat) simple image gallery written in Go. It is fast and will happily serve hundreds of galleries with a single small process.

Requirements
------------
- A working [Go][1] installation.
- A working ImageMagick installation.
- A web server to stick in front of Gollery (ideally nginx).

[1]: http://golang.org/doc/install  "Getting Started - The Go Programming Language"

Installation
------------
1. Install Gollery:

        go get github.com/madcowfred/Gollery
        cd $GOPATH/github.com/madcowfred/Gollery
        go build

2. Copy the sample config file and edit it:

        cp sample.conf gollery.conf
        vi gollery.conf

3. Run Gollery:

        ./Gollery

4. Set up some nginx vhosts:

        # Gollery is served from root
        server {
            listen 80;
            server_name images.example.com;
            
            location / {
                proxy_pass http://127.0.0.1:8089;
                proxy_set_header X-Gollery Test; # for [Gallery "Test"] in gollery.conf
                proxy_set_header X-Real-IP $remote_addr;
            }
        }
        # Gollery is not served from root
        server {
            listen 80;
            server_name www.example.com;
            
            # BaseURL should be set to /images/ for this gallery in gollery.conf
            location /images/ {
                rewrite /images(/.*) $1 break;
                proxy_pass http://127.0.0.1:8089;
                proxy_set_header X-Gollery Moo; # for [Gallery "Moo"] in gollery.conf
                proxy_set_header X-Real-IP $remote_addr;
            }
        }
