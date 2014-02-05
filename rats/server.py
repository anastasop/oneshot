#!/usr/bin/python2

import sys, random, sqlite3
import tornado.web, tornado.ioloop, tornado.template



template_text = """<DOCTYPE! html>
<html>
<head><title>Rats</title></head>
<body>
<h1>Rats</h1>
{% for image in images %}
<a href="{{image[4]}}"><img width="512" src="{{image[8]}}"/></a>
{% end %}
</body>
</html>
"""


class MainHandler(tornado.web.RequestHandler):
    def initialize(self, conn):
        self.conn = conn
        self.template = tornado.template.Template(template_text)

    def get(self):
        self.set_status(200)
        self.set_header('Content-Type', 'text/html')
        images = []
        cursor = conn.cursor()
        cursor.execute('select count(*) from images')
        number_images = cursor.fetchone()[0]
        for i in xrange(9):
            selected_id = random.randint(1, number_images)
            cursor.execute('select * from images where id = ?', (selected_id,))
            row = cursor.fetchone()
            images.append(row)
        cursor.close()
        self.write(self.template.generate(images = images))


class NotFoundHandler(tornado.web.RequestHandler):
    def get(self):
        self.send_error(404)



if __name__ == "__main__":
    conn = sqlite3.connect(sys.argv[1])

    application = tornado.web.Application([
            (r"/favicon.ico", NotFoundHandler),
            (r"/", MainHandler, dict(conn = conn))
    ])
    application.listen(9090)
    tornado.ioloop.IOLoop.instance().start()



