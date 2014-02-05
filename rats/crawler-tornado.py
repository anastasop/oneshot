#!/usr/bin/python2

import sys, base64, random, logging, sqlite3
import bs4, feedparser
import tornado.gen, tornado.httpclient, tornado.ioloop
from PIL import Image

class Storage:
    def __init__(self, name):
        self.conn = sqlite3.connect(name)
        cursor = self.conn.cursor()
        cursor.execute('''create table if not exists images(
            id integer primary key,
            feed_title text,
            feed_url text,
            story_when text,
            story_url text,
            story_title text,
            story_descr text,
            image_type text,
            image_data blob)''')
        self.conn.commit()
        cursor.close()

    def add_story(self, story):
        cursor = None
        try:
            cursor = self.conn.cursor()
            cursor.execute('insert into images values(?, ?, ?, ?, ?, ?, ?, ?, ?)', story)
            self.conn.commit()
        except sqlite3.Error as e:
            logging.error('failed to write story to database: %s', e)
        finally:
            if cursor:
                cursor.close()


class Crawler:
    def __init__(self, storage, max_entries, size_thres):
        self.url_cache = set()
        self.storage = storage
        self.max_entries = max_entries
        self.size_thres = size_thres


    @tornado.gen.coroutine
    def fetch_images(self, rss_url):
        http_client = tornado.httpclient.AsyncHTTPClient()

        # fetch the feed
        rss_xml = None
        try:
            resp = yield http_client.fetch(rss_url)
            rss_xml = resp.body
        except tornado.httpclient.HTTPError as e:
            logging.error('failed to retrieve rss from %s: reason:  %s', rss_url, e)
            return

        # parse feed
        feed = feedparser.parse(rss_xml)
        feed_title = feed.feed.title
        feed_url = rss_url
        logging.info('got feed with title %s and %d entries', feed_title, len(feed.entries))

        # fetch images from the entries
        for entry in feed.entries[0: self.max_entries]:
            try:
                resp = yield http_client.fetch(entry.link)
                entry_html = resp.body
                soup = bs4.BeautifulSoup(entry_html)
                images = soup.find_all('img')
                for img in images:
                    img_url = None
                    if "src" in img.attrs:
                        img_url = img["src"]
                    if img_url and img_url not in self.url_cache and img_url.startswith('http://'):
                        resp = yield http_client.fetch(img["src"])
                        image = Image.open(resp.buffer)
                        self.url_cache.add(img_url)
                        if image.size < self.size_thres:
                            logging.debug("rejected image %s because of small size %s'", img_url, image.size)
                        else:
                            image_type = unicode(resp.headers['Content-Type'])
                            image_data = u'data:' + image_type + u';base64,' + base64.b64encode(resp.body)
                            pt = entry.published_parsed
                            story = (None, feed_title, feed_url, sqlite3.Timestamp(pt[0], pt[1], pt[2], pt[3], pt[4], pt[5]),
                                     entry.link, entry.title, entry.description, image_type, image_data)
                            self.storage.add_story(story)
            except tornado.httpclient.HTTPError as e:
                logging.error('http error: reason: %s', e)
            except Exception as e:
                logging.error('general error %s', e)


@tornado.gen.coroutine
def main():
    storage = Storage(sys.argv[1])
    crawler = Crawler(storage, 30, (256, 256))

    rss_list = []
    with open(sys.argv[2]) as rss_file:
        for rss in rss_file:
            rss_list.append(rss)

    futures = []
    for rss in rss_list:
        futures.append(crawler.fetch_images(rss))
    yield futures




if __name__ == '__main__':
    logging.getLogger().setLevel(logging.INFO)
    tornado.ioloop.IOLoop.instance().run_sync(main)
