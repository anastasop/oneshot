#!/usr/bin/python

import SocketServer, BaseHTTPServer, urlparse, os

#class HttpServer(SocketServer.ThreadingMixIn, BaseHTTPServer.HTTPServer):
class HttpServer(BaseHTTPServer.HTTPServer):
	pass


class HttpHandler(BaseHTTPServer.BaseHTTPRequestHandler):
	def do_GET(self):
		t = urlparse.urlparse(self.path)
		f = urlparse.parse_qs(t.query, True)
		op = 'none'
		if 'op' in f:
			op = f['op'][0]
		self.log_message("url: %s operation: %s", self.path, op)
		self.send_response(201)
		self.send_header('Content-Type', 'text/plain')
		self.end_headers()
		self.wfile.write('OK')
		if op == 'stop':
			self.log_message('exiting on user request')
			os._exit(2)


if __name__ == '__main__':
	server = HttpServer(('', 8080), HttpHandler)
	server.allow_reuse_address = True
	server.serve_forever()
