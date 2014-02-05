#!/usr/bin/python2

import sys, cgi, BaseHTTPServer
import pystache, feedparser, requests
from bs4 import BeautifulSoup


RSS_SUBMIT=u"""<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8"/>
  <title>Rats Monitor</title>
</head>
<body>
<h1>Name the rats!</h1>
<hr/>
<pre>
http://cdn1-parismatch.ladmedia.fr/var/exports/rss/rss.xml
http://cdn1-parismatch.ladmedia.fr/var/exports/rss/rss-actu.xml
http://cdn1-parismatch.ladmedia.fr/var/exports/rss/rss-chroniques.xml
</pre>
<hr/>
<form action="/monitor" method="post">
  <label for="link">RSS Rats</label>
  <input type="text" size="80" id="link" name="link"/>
  <input type="submit" value="monitor"/>
</form>
</body>
</html>
"""

RSS_VIEW=u"""<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8"/>
  <title>Rats Monitor</title>
</head>
<body>
<h1>{{title}}</h1>
<hr/>
{{#entries}}
<div class="story">
<a href="{{url}}">{{title}}</a>
{{#image}}<div class="story_image"><img src="{{.}}"/></div>{{/image}}
</div>
{{/entries}}
</body>
</html>
"""

#class HttpServer(SocketServer.ThreadingMixIn, BaseHTTPServer.HTTPServer):
class HttpServer(BaseHTTPServer.HTTPServer):
    pass


class HttpHandler(BaseHTTPServer.BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.send_header('Content-Type', 'text/html')
        self.end_headers()
        self.wfile.write(RSS_SUBMIT.encode('utf-8'))


    def largest_image_in_page(self, url):
        resp = requests.get(url)
        if resp.status_code != requests.codes.ok:
            return None
        page_html = resp.content
        soup = BeautifulSoup(page_html)
        images = soup.find_all("img")
        max_area = 0
        selected_url = None
        for img in images:
            this_area = 0
            if "width" in img.attrs and "height" in img.attrs:
                this_area = int(img["width"]) * int(img["height"])
            if this_area > max_area:
                max_area = this_area
                selected_url = img["src"]
        return selected_url


    def do_POST(self):
        len = int(self.headers.getheader('Content-Length'))
        qs = self.rfile.read(len)
        form = cgi.parse_qs(qs)
        if "link" not in form:
            self.send_response(400)
            self.send_header('Content-Type', 'text/plain')
            self.end_headers()
            self.wfile.write("bad request. No RSS link found")
            return

        rss_url = form["link"][0]
        rss_resp = requests.get(rss_url)
        if rss_resp.status_code != requests.codes.ok:
            self.send_response(400)
            self.send_header('Content-Type', 'text/plain')
            self.end_headers()
            self.wfile.write("bad link '{}'. Response code {}".format(rss_url, rss_resp.status_code))
            return

        rss_xml = rss_resp.content
        feed = feedparser.parse(rss_xml)
        entries = []
        for entry in feed.entries[0:20]:
            self.log_message("filtering %s", entry.link)
            image_url = self.largest_image_in_page(entry.link)
            entries.append({"title": entry.title, "url": entry.link, "image": image_url})

        view = pystache.render(RSS_VIEW, { "title": feed.feed.title, "entries": entries })
        self.send_response(200)
        self.send_header('Content-Type', 'text/html')
        self.end_headers()
        self.wfile.write(view.encode('utf-8'))




if __name__ == '__main__':
    server = HttpServer(('', 8080), HttpHandler)
    server.allow_reuse_address = True
    server.serve_forever()
