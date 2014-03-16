/* slowcat, Copyright (c) 2000-2013 Jamie Zawinski <jwz@dnalounge.com>
 *
 * Usage: slowcat [ --verbose ]
 *                [ --debug ]
 *                [ --bps bits-per-second ]
 *                [ --burst seconds ]
 *                [ --range from-byte [ to-byte ] ]
 *                [ --icy-interval bytes ]
 *                [ --id3 ]
 *                [ [ --title string ] filename ] ...
 *
 *   Copies the given files to stdout, throtting the rate of output to be
 *   approximately (but not less than) the given bit rate (default "128k").
 *
 *   If --burst is specified, the first N seconds worth of data will be
 *   streamed at full, unthrottled speed, to fill player buffers quickly.
 *
 *   If a byte range is given, only those bytes are copied.  The byte range
 *   is across all the files, starting with byte 0 of the first file.
 *
 *   If --icy-interval is specified, Shoutcast-style metadata will be written
 *   every N bytes.
 *
 *   If --id3 is specified, an ID3v2.3 "TIT2" tag will be emitted at the
 *   beginning of each file.
 *
 *   The --title option specifies the title to be used for Icy or ID3 for
 *   each file, defaulting to the file name.
 *
 * Permission to use, copy, modify, distribute, and sell this software and its
 * documentation for any purpose is hereby granted without fee, provided that
 * the above copyright notice appear in all copies and that both that
 * copyright notice and this permission notice appear in supporting
 * documentation.  No representations are made about the suitability of this
 * software for any purpose.  It is provided "as is" without express or 
 * implied warranty.
 *
 * Created: 10-Jun-2000
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <time.h>
#include <sys/time.h>
#include <sys/errno.h>
#include <fcntl.h>
#include <unistd.h>

char *progname;
int verbose_p = 0;
int debug_p = 0;

typedef struct {
  char *title;
  const char *filename;
  struct stat st;
  int id3_size;		/* Bytes taken up by generated ID3 tag */
  int icy_size;		/* Bytes taken up by all generated ICY data */
} slowcat_file;


typedef struct {
  char *metadata, *prev_metadata;
  int icyp, id3p;
  int out_fd;
  int bps;
  char *data;
  int size;
  int fp;

  int batch_written;   /* for bps delay computation */
  int total_written;   /* for debug statistics */
  int content_length;  /* actual number of bytes sent to the stream */
  int burst_remaining; /* are we still in burst mode? */
  time_t start, batch_start, last_stats;

} slowcat_data;



static void
my_usleep (unsigned long usecs)
{
  struct timeval tv;
  tv.tv_sec  = usecs / 1000000L;
  tv.tv_usec = usecs % 1000000L;
  (void) select (0, 0, 0, 0, &tv);
}


static void
write_all (int fd, const char *buf, size_t count)
{
  while (count > 0)
    {
      int n = write (fd, buf, count);
      if (n < 0)
        {
          char buf2[1024];
          if (errno == EINTR || errno == EAGAIN)
            continue;

          sprintf (buf2, "%.255s: write:", progname);
          perror (buf2);
          exit (1);
        }

      count -= n;
      buf += n;
    }
}


static void
print_stats (slowcat_data *scd)
{
  int secs, rate;
  double off, frate;
  time_t now = time ((time_t *) 0);
  secs = now - scd->start;

  if (secs == 0)
    {
      fprintf (stderr, "%s: wrote %d bytes in one chunk.\n",
               progname, scd->total_written);
    }
  else
    {
      frate = scd->total_written * 8 / (double) secs;
      rate = frate;
      off = frate / (scd->bps * 8);

      fprintf (stderr,
               "%s: actual bits per second: %d = %.1fk (%d bytes in %ds)\n",
               progname, rate, frate / 1024, scd->total_written, secs);

      if (off < 1.0)
        fprintf (stderr, "%s: undershot by %.1f%%\n", progname,
                 (1.0 - off) * 100);
      else if (off > 1.0)
        fprintf (stderr, "%s: overshot by %.1f%%\n", progname,
                 -(1.0 - off) * 100);
    }
}


static void
slowcat_buffer (slowcat_data *scd, char *in_buf, int in_bufsiz)
{
  while (in_bufsiz > 0)
    {
      int n = in_bufsiz;
      int left = scd->size - scd->fp;
      if (n > left) n = left;
      memcpy (scd->data + scd->fp, in_buf, n);
      scd->fp   += n;
      in_buf    += n;
      in_bufsiz -= n;

      if (scd->fp > scd->size) abort();
      if (scd->fp == scd->size)
        {
          /* Buffer is full: time to write it and the current metadata.
           */
          int wrote = scd->fp;
          write_all (scd->out_fd, scd->data, scd->fp);

          scd->batch_written  += scd->fp;
          scd->total_written  += scd->fp;
          scd->content_length += scd->fp;
          scd->fp = 0;

          if (scd->metadata && scd->icyp)
            {
              int L;
              char *md = (char *) calloc (1, strlen(scd->metadata) + 200);

              if (*scd->metadata)
                {
                  sprintf (md, "\001StreamTitle='%s';", scd->metadata);
                  L = strlen(md+1) + 2;
                  L = 16 * ((L + 16) / 16);	/* round to 16 */
                  md[0] = (char) (L / 16);	/* mod16 in byte 0 */

                  if (verbose_p)
                    fprintf (stderr, "%s: ICY [%d]: %s\n", 
                             progname, L, md+1);
                }
              else
                {
                  L = 0;			/* no-op metadata */
                  *md = 0;
                }

              write_all (scd->out_fd, md, L+1);

              /* Don't count these when computing bitrate:
                 scd->batch_written += L+1;
                 scd->total_written += L+1;
               */
              scd->content_length += L+1;
              wrote += L+1;
              free (md);

              /* Nuke this metadata string so that it is only written once. */
              *scd->metadata = 0;
            }

          if (verbose_p)
            fprintf (stderr, "%s: wrote %d bytes%s\n", progname, wrote,
                     (scd->burst_remaining > 0 ? " burst" : ""));

          if (scd->burst_remaining > 0)
            {
              scd->burst_remaining -= wrote;
              if (scd->burst_remaining < 0)
                scd->burst_remaining = 0;

              /* don't sleep waiting for the burst to drain. */
              scd->batch_written = 0;

              /* don't count the burst in bps computation in stats. */
              scd->total_written = 0;
            }
        }

      /* If we've written a bunch of bytes, maybe it's time to sleep.
       */

      if (!debug_p &&
          !scd->burst_remaining &&
          scd->batch_written >= scd->bps)
        {
          /* how many seconds the batch we just wrote should have taken. */
          int secs = scd->batch_written / scd->bps;
          time_t now = time ((time_t *) 0);

          /* Wait for the second to tick. */
          while (now < scd->batch_start + secs)
            {
              if (now < scd->batch_start - 1)
                sleep (now - scd->batch_start - 1);	/* N-1 seconds */
              else
                my_usleep (20000L);			/* 1/50th second */

              now = time ((time_t *) 0);
            }

          /* Second has ticked, restart counter and start writing again. */
          scd->batch_start = now;
          scd->batch_written = 0;

          if (verbose_p)
            {
              if (scd->last_stats + 10 <= now)
                {
                  print_stats (scd);
                  scd->last_stats = now;
                }
            }
        }
    }
}


static void
write_id3v2_title (slowcat_data *scd, const char *title)
{
# define ID3_SIZE 1214
  char buf [ID3_SIZE];
  int head_len = 10;
  int tag_len = sizeof(buf) - head_len;
  int title_len;
  char *t2;
  int i;

  if (!title || !*title) return;
  t2 = strdup (title);
  title_len = strlen (title);

  if (title_len > 1190)  /* Just truncate insanely long ones */
    {
      title_len = 1190;
      t2[title_len] = 0;
    }

  if (verbose_p)
    fprintf (stderr, "%s: ID3 [%lu]: %s\n", progname, sizeof(buf), t2);

  title_len++;  /* include null */

  i = 0;
  memset (buf, 0, sizeof(buf));
  strncpy (buf+i, "ID3", 3);		/* 3: frame type "ID3" */
  i += 3;
  buf[i++] = 3;				/* 2: ID3v2.3 version number */
  buf[i++] = 0;
  buf[i++] = 0;				/* 1: flags */
  buf[i++] = (tag_len >> 21 & 0x7F);	/* 4: crazy 7-bit packing */
  buf[i++] = (tag_len >> 14 & 0x7F);
  buf[i++] = (tag_len >>  7 & 0x7F);
  buf[i++] = (tag_len       & 0x7F);
  strncpy (buf+i, "TIT2", 4);		/* 4: tag name "TIT2" */
  i += 4;
  buf[i++] = (title_len >> 24 & 0xFF);	/* 4: title length */
  buf[i++] = (title_len >> 16 & 0xFF);
  buf[i++] = (title_len >>  8 & 0xFF);
  buf[i++] = (title_len       & 0xFF);
  buf[i++] = 0;				/* 2: flags */
  buf[i++] = 0;
  buf[i++] = 0;				/* 1: charset */
  strncpy (buf+i, t2, title_len);	/* title including null */
  free (t2);

  slowcat_buffer (scd, buf, sizeof(buf));
}


static void
slowcat_one_file (slowcat_data *scd, int from, int to, slowcat_file *file)
{
  static char buf[100 * 1024];
  int bufsiz = sizeof(buf)-1;
  int total_read = 0;
  int id3p = scd->id3p;
  int n;

  int in_fd = open (file->filename, O_RDONLY);

  if (in_fd < 0)
    {
      sprintf (buf, "%.255s: %.255s", progname, file->filename);
      perror (buf);
      exit (1);
    }

  if (scd->metadata) free (scd->metadata);
  scd->metadata = strdup (file->title);

  if (verbose_p)
    fprintf (stderr, "%s: streaming: %s\n", progname, file->filename);

  if (from > 0)
    {
      if (from != lseek (in_fd, from, SEEK_SET))
        {
          sprintf (buf, "%.255s: %.255s: seek:", progname, file->filename);
          perror (buf);
          exit (1);
        }
    }

  if (to >= 0)
    to -= from;

  while ((n = read (in_fd, buf, bufsiz)) != 0)
    {
      if (n < 0)
        {
          if (errno == EINTR || errno == EAGAIN)
            continue;

          sprintf (buf, "%.255s: read", progname);
          perror (buf);
          exit (1);
        }

      if (to >= 0 && total_read + n > to)
        {
          if (verbose_p)
            fprintf (stderr, "%s: %s: range end\n", progname, file->filename);
          n = to - total_read;
          if (n < 0) abort();
          if (n == 0) break;
        }

      if (id3p)
        {
          write_id3v2_title (scd, file->title);
          id3p = 0;  /* only once */
        }

      slowcat_buffer (scd, buf, n);
      total_read += n;
    }

  close (in_fd);
}


static void
slowcat_all_files (int bps, int burst, int icy_interval, int id3p,
                   int from, int to, 
                   int nfiles, slowcat_file *files)
{
  slowcat_data SCD;
  slowcat_data *scd = &SCD;
  int i = 0;
  int bytes_written = 0;

  memset (scd, 0, sizeof(*scd));
  scd->out_fd = fileno(stdout);
  scd->bps = bps / 8;         /* bytes per second, not bits per second. */
  scd->size = (icy_interval ? icy_interval : scd->bps * 2);
  scd->data = (char *) calloc (1, scd->size);
  scd->start = time ((time_t *) 0);
  scd->batch_start = scd->start;
  scd->last_stats = scd->start;
  scd->icyp = (icy_interval > 0);
  scd->id3p = id3p;
  scd->burst_remaining = burst * scd->bps;

  for (i = 0; i < nfiles; i++)
    {
      int size = files[i].st.st_size;

      if ((to >= 0 && bytes_written > to) ||
          (bytes_written + size <= from))
        {
          if (verbose_p)
            fprintf (stderr, "%s: skipping file %s (%d)\n",
                     progname, files[i].filename, size);
        }
      else
        {
          int from2 = from - bytes_written;
          int to2   = to   - bytes_written;
          if (from2 < 0)    from2 = 0;
          if (to2   > size) to2   = size;
          slowcat_one_file (scd, from2, to2, &files[i]);
        }
      bytes_written += size;
    }

  if (scd->fp > 0)    /* Flush the last buffer */
    {
      write_all (scd->out_fd, scd->data, scd->fp);
      if (verbose_p)
        fprintf (stderr, "%s: wrote %d bytes [EOF]\n", progname, scd->fp);
      scd->content_length += scd->fp;
    }

  if (verbose_p)
    {
      if (from)
        fprintf (stderr, "%s: skipped %d bytes\n", progname, from);
      fprintf (stderr, "%s: actual content length: %d\n", progname, 
               scd->content_length);
    }
}


char *
filename_to_title (const char *filename)
{
  char *title = malloc (strlen(filename) * 2 + 1);
  char *s, *out = title;

  s = strrchr (filename, '/');			/* lose directory */
  if (s) filename = s+1;

  while (*filename >= '0' && *filename <= '9')  /* skip leading numbers */
    filename++;
  while (*filename == ' ')                      /* skip leading spaces */
    filename++;

  while (*filename)
    {
      char c = *filename++;
      *out++ = c;
    }
  *out = 0;

  s = strrchr (title, '.');
  if (s) *s = 0;				/* truncate before ".mp3" */

  return title;
}


static void
usage (const char *err)
{
  if (err)
    fprintf (stderr, "%s: %s\n", progname, err);
  fprintf (stderr, "usage: %s\t[ --verbose ]\n\
		[ --debug ]\n\
		[ --bps bits-per-second ]\n\
		[ --burst seconds ]\n\
		[ --range from-byte [ to-byte ] ]\n\
		[ --icy-interval bytes ]\n\
		[ --id3 ]\n\
		[ [ --title string ] filename ] ...\n\n",
           progname);
  exit (1);
}


int
main (int argc, char **argv)
{
  char *s, c;
  int i;
  int bps = 128 * 1024;
  int burst = 0, from = 0, to = -1, icy_interval = 0, id3p = 0;
  const char *title = 0;
  slowcat_file *files;
  int nfiles = 0;

  progname = argv[0];
  s = strrchr (progname, '/');
  if (s) progname = s+1;

  files = (slowcat_file *) calloc (argc, sizeof(*files));

  for (i = 1; i < argc; i++)
    {
      const char *arg = argv[i];
      if (arg[0] == '-' && arg[1] == '-')
        arg++;

      if (!strcmp (arg, "-verbose"))   verbose_p++;
      else if (!strcmp (arg, "-v"))    verbose_p++;
      else if (!strcmp (arg, "-vv"))   verbose_p += 2;
      else if (!strcmp (arg, "-vvv"))  verbose_p += 3;
      else if (!strcmp (arg, "-vvvv")) verbose_p += 4;
      else if (!strcmp (arg, "-debug")) debug_p++;
      else if (!strcmp (arg, "-d"))     debug_p++;
      else if (!strcmp (arg, "-bps"))
        {
          i++;
          if (i >= argc) usage("no bps");
          if (1 != sscanf (argv[i], "%d%c", &bps, &c))
            {
              if (1 != sscanf (argv[i], "%dk%c", &bps, &c) &&
                  1 != sscanf (argv[i], "%dK%c", &bps, &c))
                usage("unparsable bps");
              bps *= 1024;
            }
          if (bps < 8 || bps > (1024 * 1024 * 1024))
            usage("insane bitrate");
        }
      else if (!strcmp (arg, "-burst"))
        {
          i++;
          if (i >= argc) usage("no burst");
          if (1 != sscanf (argv[i], "%d%c", &burst, &c))
            usage("unparsable burst");

        }
      else if (!strcmp (arg, "-range"))
        {
          i++;
          if (i >= argc) usage("no range");
          if (1 != sscanf (argv[i], "%d%c", &from, &c))
            usage("unparsable range start");

          if (i+1 < argc &&
              1 == sscanf (argv[i+1], "%d%c", &to, &c))
            i++;

        }
      else if (!strcmp (arg, "-icy-interval") ||
               !strcmp (arg, "-icy"))
        {
          i++;
          if (i >= argc) usage("no icy");
          if (1 != sscanf (argv[i], "%d%c", &icy_interval, &c))
            usage("unparsable icy");

        }
      else if (!strcmp (arg, "-id3"))
        {
          id3p = 1;
        }
      else if (!strcmp (arg, "-title"))
        {
          i++;
          if (i >= argc) usage("no title");
          title = argv[i];
          if (title[0] == '-') usage("titles don't begin with dash");

        }
      else if (arg[0] == '-')
        {
          char buf[1024];
          sprintf (buf, "unknown argument: %.255s", arg);
          usage(buf);
        }
      else
        {
          files[nfiles].filename = arg;

          files[nfiles].title = 
            (title
             ? strdup (title)
             : filename_to_title (files[nfiles].filename));
          title = 0;

          if (0 > stat(files[nfiles].filename, &files[nfiles].st))
            {
              char buf[1024];
              sprintf (buf, "%.255s: %.255s: fstat:", progname,
                       files[nfiles].filename);
              perror (buf);
              exit (1);
            }

          if (id3p)
            files[nfiles].id3_size = ID3_SIZE;
          if (icy_interval)
            {
              /* Length of the StreamTitle='' metadata */
              int L = strlen (files[nfiles].title) + 17;
              L = 16 * ((L + 16) / 16);		/* round to 16 */

              /* Add in one byte per buffer, for subsequent blank metadata */
              L += (files[nfiles].st.st_size / icy_interval);

              files[nfiles].icy_size = L;
            }

          nfiles++;
        }
    }

  if (nfiles == 0) usage("no files");

  if (to >= 0 && to - from <= ID3_SIZE)
    id3p = 0;	/* if we're asking for a tiny range, omit the ID3 tag. */

  slowcat_all_files (bps, burst, icy_interval, id3p, from, to, nfiles, files);

  return 0;
}
