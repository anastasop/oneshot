#include <sys/types.h>
#include <sys/stat.h>
#include <fcntl.h>
#include <unistd.h>
#include <stdio.h>
#include <stdlib.h>
 
/*
  measure the time it takes to write a disk file
  using synchronized writes.
 
  usage: time ./a.out -c <chunksize> -n <nchunks> [-a]
 
  writes nchunks(default 1024) of chunksize(default 1024) bytes each
  if -a is enabled then it does not sync (O_SYNC)
  after every write but uses the OS page cache.
  With -a we expect faster times.
 
  This is a micro benchmark to demonstrate that if you
  want a fully reliable system with 0 data loss there is
  a penalty to pay. disks dominate time and QPS are low
*/
 
int
main(int argc, char **argv)
{
	int i, fd, arg, nchunks, chunksize, async;
	ssize_t n;
	char *chunk;
	mode_t fmode;
 
	chunksize = 1024; // default 1K
	nchunks = 1024; // default file output 1M (nchunks * chunksize)
	async = 0;
	while ((arg = getopt(argc, argv, "n:c:a")) != -1) {
		switch(arg) {
		case 'n':
			nchunks = atoi(optarg);
			break;
		case 'c':
			chunksize = atoi(optarg);
			break;
		case 'a':
			async = 1;
			break;
		}
	}
	chunk = (char*)malloc(chunksize);
	if (chunk == NULL) {
		perror("malloc failed");
		exit(2);
	}
	for (i = 0; i < chunksize; i++) {
		chunk[i] = rand() % 256;
	}
 
	fmode = O_WRONLY | O_APPEND | O_CREAT | O_TRUNC | O_APPEND;
	if (!async) {
		fmode |= O_SYNC;
	}
	fd = open("./chunks.dat", fmode, 0644);
	if (fd < 0) {
		perror("open failed");
		exit(2);
	}
 
	for (i = 0; i < nchunks; i++) {
		n = write(fd, chunk, chunksize);
		if (n < 0) {
			perror("write failed");
			exit(2);
		}
		if (n != chunksize) {
			// with local disks (no NFS) this happens very rarely if not at all 
			fprintf(stderr, "warning: written less than a chunk: %d\n", n);
		}
	}
	if (close(fd) < 0) {
		perror("close failed");
		exit(2);
	}
	return 0;
}
