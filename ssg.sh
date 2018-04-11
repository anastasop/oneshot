#!/bin/sh
#
# https://www.romanzolotarev.com/bin/ssg
# Copyright 2018 Roman Zolotarev
#
# Permission to use, copy, modify, and/or distribute this software for any
# purpose with or without fee is hereby granted, provided that the above
# copyright notice and this permission notice appear in all copies.
#
# THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
# WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
# MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
# ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
# WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
# ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
# OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
#
# set -e

usage() {
  echo 'usage: export HOST=127.0.0.1'
  echo '       export PORT=4000'
  echo '       export DOCS=docs'
  echo
  echo '       ssg'
  echo '         | build'
  echo '         | watch'
  echo '         | serve'
  exit 1
}

copy_files() {
  [ "$(dirname "$1")" = "$PWD" ] && self="/$(basename "$1")/" || self="$1"
  rsync -a "." "$1" --delete-excluded \
    --exclude "$self" \
    --exclude ".*" \
    --exclude "_*" \
    --exclude "node_modules"
  echo '<h1>Sitemap</h1><table>' > "$1/sitemap.html"
}

md_to_html() {
  find "$1" -type f -name '*.md'|while read -r file; do
    lowdown -D html-skiphtml "$file" > "${file%\.md}.html" && rm "$file"
  done
}

wrap_html() {
  #
  # generate sorted sitemap
  sitemap="$(find "$1" -type f -name '*.html'|while read -r file; do
    awk '/<[h1]*( id=".*")?>/{gsub(/<[^>]*>/,"");print(FILENAME";"$0)}' "$file"
  done|sort)"
  #
  # save sitemap in html and xml formats
  date=$(date +%Y-%m-%dT%H:%M:%S%z)
  echo '<?xml version="1.0" encoding="UTF-8"?><urlset xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.sitemaps.org/schemas/sitemap/0.9 http://www.sitemaps.org/schemas/sitemap/0.9/sitemap.xsd" xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">' > "$1/sitemap.xml"
  echo "$sitemap"|while read -r line; do
    page=${line%;*}
    url=${page#$1}
    case "$url" in
      /index.html) title='Home';;
      *) title="${line#*;}";;
    esac
    echo "<tr><td><a href=\"$url\">$url</a><td> $title</tr>" >> "$1/sitemap.html"
    echo "<url><loc>https://www.romanzolotarev.com$url</loc><lastmod>$date</lastmod><priority>1.0</priority></url>" >> "$1/sitemap.xml"
  done
  echo '</table>' >> "$1/sitemap.html"
  echo '</urlset>' >> "$1/sitemap.xml"
  #
  # generate html pages
  echo "$sitemap"|while read -r line; do
    page=${line%;*}
    url=${page#$1}
    article=$(cat "$page")
    case "$url" in
      /index.html)
        title='Home'
        head_title='<title>Roman Zolotarev</title>'
        header__home='Home'
        ;;
      *)
        title="${line#*;}"
        head_title="<title>$title - Roman Zolotarev</title>"
        header__home='<a href="/">Home</a> - <a href="https://twitter.com/romanzolotarev">Twitter</a>'
        ;;
    esac
    # merge page with html template
    cat > "$page" <<EOF
<!DOCTYPE html><html lang="en">
<head>$head_title<meta charset="utf-8"><meta name="referrer" content="no-referrer"><link rel="stylesheet" href="/styles.css"><link rel="manifest" href="/manifest.webmanifest"><meta name="theme-color" content="#ffffff"><meta http-equiv="x-ua-compatible" content="ie=edge"><meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="apple-mobile-web-app-capable" content="yes">
<meta name="apple-mobile-web-app-status-bar-style" content="black">
</head>
<body><script>!function(t){ t.addEventListener('DOMContentLoaded', function () { var l = t.querySelector('#light-off'); if (l === null) { console.log('Lights-out...'); } else { l.checked = t.cookie.match(/lightOff=true/) !== null; l.addEventListener('change', function () { t.cookie = 'lightOff=' + JSON.stringify(l.checked) + ';path=/'; }); } })}(document);</script><input class="light-off" type="checkbox" id="light-off">
<div class="page">
<div class="header"><div class="header__home">$header__home</div><div class="header__light-off"><label for="light-off" class="light-off-button"></label></div></div>
<div class="article">$article</div>
<div class="footer">&copy; <a href="/">Roman Zolotarev</a></div>
</div></body></html>
EOF
  done
  echo "$date $(echo "$sitemap"|wc -l|tr -d ' ')pp"
}

[ -z "$DOCS" ] && DOCS=$(readlink -fn "./docs") || DOCS=$(readlink -fn "$DOCS")

case "$1" in

build)
  which rsync >/dev/null 2>&1 || { echo 'rsync(1) should be installed'; exit 1; }
  which lowdown >/dev/null 2>&1 || { echo 'lowdown(1) should be installed'; exit 1; }
  copy_files "$DOCS"
  md_to_html "$DOCS"
  wrap_html "$DOCS"
  ;;

watch)
  echo "watching $PWD"
  which entr >/dev/null 2>&1 || { echo 'entr(1) should be installed'; exit 1; }
  while true; do
    find "$PWD" -type f \
      \( -name "$(basename "$0")" -or -name '*.md' -or -name '*.html' -or -name '*.css' -or -name '*.txt' \)\
      ! -name ".*" ! -path "*/.*" ! -path "*/node_modules*" ! -path "${DOCS}*" |
      entr -d ssg build
  done
  ;;

serve)
  [ -z "$PORT" ] && PORT='4000'
  [ -z "$HOST" ] && HOST='127.0.0.1'
  which httpd >/dev/null 2>&1 || { echo 'httpd(8) should be installed'; exit 1; }
  logdir="$DOCS/logs"
  conf="$DOCS/httpd.conf"
  echo "serving $DOCS"
  echo "listening http://$HOST/$PORT"
  mkdir -p "$DOCS" "$logdir"
  cat > "$conf" <<EOF
chroot "$DOCS"
logdir "$logdir"
server "$HOST" {
  listen on $HOST port $PORT
  root "/"
}
EOF
  cd "$DOCS" || exit
  doas httpd -d -f "$conf"
  ;;

'')
  ssg serve &
  ssg watch
  ;;

*) usage;;

esac
