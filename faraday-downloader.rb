
require 'zlib'
require 'stringio'
require 'faraday'
require 'faraday_middleware'
require 'zip'

module Moracle
  CONTENT_TYPE = 'Content-Type'.freeze

  class DownloaderError < StandardError; end


  class GzipTransferEncoding < Faraday::Response::Middleware
    def on_complete(env)
      encoding = env[:response_headers]['Content-Encoding'].to_s.downcase
      case encoding
      when 'gzip'
        env[:body] = Zlib::GzipReader.new(StringIO.new(env[:body])).read
        env[:response_headers].delete('Content-Encoding')
      when 'deflate'
        env[:body] = Zlib::Inflate.inflate(env[:body])
        env[:response_headers].delete('Content-Encoding')
      end
    end
  end


  class ContentLengthLimiter < Faraday::Response::Middleware
    MAXIMUM_SIZE_BODY = 300 * 1048576

    def on_complete(env)
      raise DownloaderError, "Request body too large: #{env[:body].length}" if env[:body].length > MAXIMUM_SIZE_BODY
    end
  end
  

  class ContentTypeSniffer < Faraday::Response::Middleware
    ZIP_MAGIC = "\x50\x4B\x03\x04".force_encoding("BINARY")
    GZIP_MAGIC = "\x1F\x8B\x08".force_encoding("BINARY")

    def on_complete(env)
      return unless env[:response_headers][CONTENT_TYPE] == 'application/octet-stream'
      sniffed = env[:body][0...6].force_encoding("BINARY")
      if sniffed.start_with? GZIP_MAGIC 
        env[:response_headers][CONTENT_TYPE] = 'application/x-gzip'
      elsif sniffed.start_with? ZIP_MAGIC
        env[:response_headers][CONTENT_TYPE] = 'application/zip'
      end
    end
  end


  class CompressedResponse < Faraday::Response::Middleware
    def on_complete(env)
      content_type = env[:response_headers][CONTENT_TYPE]
      case content_type
      when "application/x-gzip", "application/gzip"
        env[:body] = Zlib::GzipReader.new(StringIO.new(env[:body])).read
        env[:response_headers][CONTENT_TYPE] = 'application/xml'
      when "application/zip", "application/x-zip-compressed"
        Zip::InputStream.open(StringIO.new(env[:body])) do |io|
          if entry = io.get_next_entry
            env[:body] = io.read
            env[:response_headers][CONTENT_TYPE] = 'application/xml'
          end
          raise DownloaderError, "More than one entries in the zip file" if io.get_next_entry
        end
      end
    end
  end


  class Downloader
    def initialize(bot_name = 'mybot')
      @conn = Faraday.new do |conn|
        conn.use CompressedResponse
        conn.use ContentTypeSniffer
        conn.use ContentLengthLimiter
        conn.use GzipTransferEncoding
        conn.use FaradayMiddleware::Chunked
        conn.use Faraday::Response::RaiseError
        conn.use FaradayMiddleware::FollowRedirects
        conn.adapter Faraday.default_adapter
      end
      @name = bot_name
    end

    def download(url)
      response = @conn.get do |req|
        req.headers['Accept-Encoding'] = 'gzip'
        req.headers['User-Agent'] = @name
        req.url url
      end
      response.body
    end
  end
end
