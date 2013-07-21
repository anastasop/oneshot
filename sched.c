
#include <sys/types.h>
#include <sys/wait.h>
#include <unistd.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <ctype.h>
#include <errno.h>

#define NELEMS(x) (sizeof(x) / sizeof(x[0]))

void
execcmd(char *cmd)
{
	char *t, *argv[32];
	int argc, ret, i;
	pid_t pid;

	t = cmd;
	argc = 0;
	do {
		while (*t && isspace(*t)) {
			t++;
		}
		if (*t) {
			argv[argc++] = t;
		}
		while (*t && !isspace(*t)) {
			t++;
		}
		if (*t) {
			*t++ = '\0';
		}
	} while (*t);
	argv[argc] = 0;

/*
	for (i = 0; i < argc; i++) {
		fprintf(stdout, "'%s'\n", argv[i]);
	}
	fprintf(stdout, "\n");
*/

	if (argc > 0) {
		pid = fork();
		switch(pid) {
		case 0:
			ret = execv(argv[0], argv);
			if (ret == -1) {
				fprintf(stderr, "exec: %s for '%s'\n", strerror(errno), cmd);
				exit(2);
			}
			break;
		case -1:
			fprintf(stderr, "fork: %s for '%s'\n", strerror(errno), cmd);
			break;
		default:
			break;
		}
	}
}


int
main(int argc, char *argv[])
{
	char *s, line[1024];

	fprintf(stdout, "sched started: %d\n", getpid());
	while (fgets(line, NELEMS(line), stdin) != NULL) {
		execcmd(line);
	}
	exit(0);
}
