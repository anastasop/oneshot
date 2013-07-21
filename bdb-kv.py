#!/usr/bin/python

import SocketServer, BaseHTTPServer, urlparse, bsddb

#class HttpServer(SocketServer.ThreadingMixIn, BaseHTTPServer.HTTPServer):
class HttpServer(BaseHTTPServer.HTTPServer):
	pass


class HttpHandler(BaseHTTPServer.BaseHTTPRequestHandler):
	def do_GET(self):
		t = urlparse.urlparse(self.path)
		f = urlparse.parse_qs(t.query, True)
		key = f['key'][0]
		val = f['val'][0]
		self.server.db[key] = val
		self.server.db.sync()
		self.send_response(201)
		self.send_header('Content-Type', 'text/plain')
		self.end_headers()
		self.wfile.write('OK')


if __name__ == '__main__':
	server = HttpServer(('', 8080), HttpHandler)
	server.allow_reuse_address = True
	server.serve_forever()
	server.db = bsddb.hashopen('store.bdb', 'c')
