
require 'sinatra/base'
require 'sqlite3'

class RatsServer < Sinatra::Base
  configure do
    enable :logging
    set :database_conn, SQLite3::Database.new('news')
    set :rng, Random.new
    set :images_per_gallery, 9
  end

  template :layout do
    %Q{
<DOCTYPE! html>
<html>
<head><title>Rats</title></head>
<body>
<h1>Rats</h1>
<%= yield %>
</body>
</html>}
  end

  template :gallery do
    %Q{<% @images.each do |img| %> <a href="<%= img[4] %>"><img width="256" height="256" src="<%= img[8] %>"/></a> <% end %>}
  end

  def get_random_images()
    images = []
    db = settings.database_conn
    rng = settings.rng
    n = settings.images_per_gallery
    begin
      count = db.get_first_value('select count(*) from images')
      n.times do
        r = rng.rand(count) + 1
        db.execute('select * from images where id = ?', [r]) {|row| images << row }
      end
    rescue SQLite3::Exception => e
      logger.error('failed to select images: ' + e.message)
    end
    return images
  end

  get '/favicon.ico' do
    return 404
  end

  get '/' do
    @images = get_random_images()
    erb :gallery
  end
end
