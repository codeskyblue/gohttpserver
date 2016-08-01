#!/usr/bin/env python
# coding: utf-8

import time
import hashlib

import tornado
import tornado.ioloop
import tornado.web
from tornado import gen, httpclient, ioloop
from tornado.options import define, options


define("port", default=8200, help="Run server on a specific port", type=int) 

class MainHandler(tornado.web.RequestHandler):
    def get(self):
        self.write("Hello, proxy")


class ProxyHandler(tornado.web.RequestHandler):
    http_client = httpclient.AsyncHTTPClient()

    @gen.coroutine
    def get(self, url):
        print time.strftime('%Y-%m-%d %H:%M:%S'), 'PROXY http://' + url
        response = yield self.http_client.fetch('http://'+url) #www.google.com')
        # print response.body
        if (response.error and not
                isinstance(response.error, tornado.httpclient.HTTPError)):
            self.set_status(500)
            self.write('Internal server error:\n' + str(response.error))
        else:
            self.set_status(response.code, response.reason)
            for header, v in response.headers.get_all():
                if header not in ('Content-Length', 'Transfer-Encoding', 'Content-Encoding', 'Connection'):
                    self.set_header(header, v) # some header appear multiple times, eg 'Set-Cookie'
            if response.body:         
                self.set_header('Content-Length', len(response.body))
                self.write(response.body)
        self.finish()

class PlistStoreHandler(tornado.web.RequestHandler):
    db = {}

    def post(self):
        body = self.request.body
        if len(body) > 5000:
            self.set_status(500)
            self.finish("request body too long")
        m = hashlib.md5()
        m.update(body)
        key = m.hexdigest()[8:16]
        self.db[key] = body
        self.write({'key': key})

    def get(self):
        key = self.get_argument('key')
        value = self.db.get(key)
        if value is None:
            raise tornado.web.HTTPError(404)
        self.set_header('Content-Type', 'text/xml')
        self.finish(value)


def make_app(debug=True):
    return tornado.web.Application([
        (r"/", MainHandler),
        (r"/proxy/(.*)", ProxyHandler),
        (r"/plist", PlistStoreHandler),
    ], debug=debug)


if __name__ == "__main__":
    app = make_app()
    tornado.options.parse_command_line() 
    app.listen(options.port)
    ioloop.IOLoop.current().start()
