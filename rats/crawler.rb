
require 'base64'
require 'logger'
require 'open-uri'
require 'feedzirra'
require 'sqlite3'
require 'dimensions'

MAX_ENTRIES = 2

$logger = Logger.new(STDOUT)
$logger.level = Logger::DEBUG
$logger.progname = 'rats'



class Storage
  def initialize(name)
    @db = SQLite3::Database.new name
    @db.execute %q{create table if not exists images(
            id integer primary key,
            feed_title text,
            feed_url text,
            story_when text,
            story_url text,
            story_title text,
            story_descr text,
            image_type text,
            image_data blob)}
  end

  def add_story(story)
    begin
      @db.execute 'insert into images values(?, ?, ?, ?, ?, ?, ?, ?, ?)', story
    rescue SQLite3::Exception => e
      $logger.error('failed to write story in db: ' + e.message)
    end
  end
end




def fetch_url(url, with_type = false)
  $logger.debug('fetch_url: "' + url + '"')
  begin
    open(url) do |f|
      if f.status[0] != '200'
        $logger.error('failed: fetch_url: "' + url + '"')
        return nil
      end
      g_body = f.read
      if with_type
        return [g_body, f.metas['content-type'][0]]
      else
        return g_body
      end
    end
  rescue Exception => e
    $logger.error('fetch_url, exception: ' + e.message)
    return nil
  end
end



$feeds = []
$storage = Storage.new('news')

File.foreach("rss.txt") do |line|
  rss_url = line.strip
  rss_xml = fetch_url(rss_url)
  if rss_xml
    $feeds << rss_xml
  end
end

$feeds.each do |feed_xml|
  feed = Feedzirra::Feed.parse(feed_xml)
  feed.entries[0...MAX_ENTRIES].each do |entry|
    html_text = fetch_url(entry.url)
    html_doc = Nokogiri::HTML(html_text)
    html_doc.css('img').each do |img|
      if img['src'] and img['src'].start_with?('http://')
        $logger.debug("ask image for '#{img['src']}'")
        img = fetch_url(img['src'], true)
        if img
          img_data, img_type = *img
          img_img = Dimensions(StringIO.new(img_data))
          if img_img and ((img_img.height and img_img.height < 256) or (img_img.width and img_img.width < 256))
            $logger.debug("rejecting small image #{img_img.dimensions}")
          else
            img_data_url = 'data:' + img_type + ';base64,' + Base64.strict_encode64(img_data)
            story = [nil, feed.title, feed.url, nil, entry.url, entry.title, entry.summary, img_type, img_data_url]
            $storage.add_story(story)
          end
        end
      end
    end
  end
end
