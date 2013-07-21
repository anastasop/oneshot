/*
 * Really simple shell.  Does
 *	simple commands
 *	file patterns
 *	quoting ('...' and \c)
 *	continuation with \ at end of line
 *	redirection
 *	pipelines
 *	synchronous and asynchronous execution (; and &)
 *	conditional execution (&& and ||)
 *	only built-ins are cd, path & exit
 *	nothing else
 *
 * Grammar is (roughly)
 *
 * cmd:	simple
 * |	simple | cmd
 * |	simple ; cmd
 * |	simple & cmd
 * |	simple && cmd
 * |	simple || cmd
 * simple:
 * |	simple word
 * |	simple < word
 * |	simple > word
 *
 * Bugs:
 *	looses track of children in very long (>NPID) pipelines.
 *	can't execute commands with very long (>1000 characters) names.
 *	continuation isn't really done correctly.
 */
#include <stdarg.h>
#include <stdio.h>
#include <stdlib.h>
#include <dirent.h>
#include <sys/stat.h>
#include <sys/signal.h>
#include <setjmp.h>
#include <fcntl.h>
#include <string.h>
#define	NPID	100
#define	NBUF	8192
#define	NDIR	256
#define	GLOB	((char)0x80)	/* used to mark globbing chars */
#define	EOF	(-1)
int skip;		/* suppress further action because of error or condition failure */
int success;			/* the last command waited for succeeded */
int fd;				/* input file descriptor */
char buf[NBUF];			/* input buffer */
int peekc=EOF;			/* character to reread, if not EOF */
char *bufp, *ebuf;		/* next input character, end of good characters */
char *shname;			/* our name */
char **args, **argp, **eargs;	/* command args */
char *charg, *chargp, *echarg;	/* command strings */
#define	NREDIR	2
int redir[NREDIR];		/* file descriptor mapping after fork */
int pid[NPID];
int *epid;
void nomem(void){
	fprintf(stderr, "%s: can't get memory!\n");
	exit(1);
}
void err(char *msg, ...){
	va_list args;
	char buf[1024], *out;
	va_start(args, msg);
	fprintf(stderr, "%s: ", shname);
	vfprintf(stderr, msg, args);
	va_end(args);
	skip=1;
}
/*
 * Start a new member of the arg list
 */
void arg(char *str){
	int narg;
	char **newargs;
	if(argp==eargs){
		narg=eargs-args+100;
		newargs=realloc(args, narg*sizeof(char *));
		if(newargs==0) nomem();
		argp=newargs+(argp-args);
		eargs=newargs+narg;
		args=newargs;
	}
	*argp++=str;
}
/*
 * add a character to the current argument
 */
void argchr(char c){
	int ncharg;
	char *newcharg;
	char **argpp;
	if(chargp==echarg){
		ncharg=echarg-charg+100;
		newcharg=realloc(charg, ncharg*sizeof(char));
		if(newcharg==0) nomem();
		chargp=newcharg+(chargp-charg);
		echarg=newcharg+ncharg;
		for(argpp=args;argpp!=argp;argpp++) *argpp=newcharg+(*argpp-charg);
		charg=newcharg;
	}
	*chargp++=c;
}
int readc(void){
	int n;
	if(peekc!=EOF){
		n=peekc;
		peekc=EOF;
		return n;
	}
	if(bufp==ebuf){
		n=read(fd, buf, NBUF);
		if(n<=0) return EOF;
		ebuf=buf+n;
		bufp=buf;
	}
	return *bufp++&255;
}
int nextis(int match){
	int c;
	c=readc();
	if(c==match) return 1;
	peekc=c;
	return 0;
}
char **path;
void setpath(char **p, char **ep){
	int i;
	char **pp;
	if(path){
		for(i=0;path[i];i++) free(path[i]);
		free(path);
	}
	path=malloc((ep-p+1)*sizeof(char *));
	if(path==0) nomem();
	pp=path;
	while(p!=ep){
		*pp=strdup(*p++);
		if(*pp++==0) nomem();
	}
	*pp=0;
}
void cmd(int defer, int async){
	int id, i, nwait;
	int *p;
	char retry[2002];	/* see sprintf below */
	int w;
	success=1;
	if(!skip && argp!=args){
		if(strcmp(args[0], "cd")==0){
			if(argp!=args+2) err("Usage: cd directory\n");
			else if(chdir(args[1])==-1) err("can't cd %s\n", args[1]);
		}
		else if(strcmp(args[0], "exit")==0){
			if(argp==args+1) exit(0);
			else if(argp==args+2) exit(atoi(args[1]));
			else err("Usage: exit [status]\n");
		}
		else if(strcmp(args[0], "path")==0){
			if(argp==args+1){
				printf("path");
				for(i=0;path[i];i++)
					printf(" %s", path[i]);
				printf("\n");
			}
			else
				setpath(args, argp);
		}
		else switch(id=fork()){
		case -1:
			err("can't fork\n");
			break;
		default:
			if(epid!=&pid[NPID]) *epid++=id;
			if(defer) break;
			if(!async){
				nwait=epid-pid;
				while(nwait!=0 && (id=wait(&w))!=-1){
					for(p=pid;p!=epid;p++){
						if(*p==id){
							--nwait;
							if(w!=0) success=0;
							break;
						}
					}
				}
			}
			epid=pid;
			break;
		case 0:
			for(i=0;i!=NREDIR;i++){
				if(redir[i]!=i){
					dup2(redir[i], i);
					close(redir[i]);
				}
			}
			if(async && redir[0]==0) close(0);
			arg(0);
			for(i=0;path[i];i++){
				sprintf(retry, "%.1000s/%.1000s",
					path[i], args[0]);
				execv(retry, args);
			}
			exit(1);
		}
	}
	for(i=0;i!=NREDIR;i++){
		if(redir[i]!=i){
			close(redir[i]);
			redir[i]=i;
		}
	}
	argp=args;
	chargp=charg;
	if(skip) success=0;
}
void pipeline(void){
	int pfd[2];
	if(!skip){
		if(redir[1]!=1) err("pipe and > together\n");
		else{
			pipe(pfd);
			redir[1]=pfd[1];
			cmd(1, 1);
			redir[0]=pfd[0];
		}
	}
}
/*
 * Returns a pointer to the remainder of the pattern, or 0 if no match
 */
char *match(char *s, char *p){
	char *np;
	for(;*p!='/' && *p!='\0';s++,p++){
		if(*p==GLOB) switch(*++p){
		case GLOB:
			if(*s!=GLOB) return 0;
			break;
		case '*':
			for(;;){
				np=match(s, p+1);
				if(np) return np;
				if(*s=='\0') break;
				s++;
			}
			return 0;
		case '?':
			if(*s=='\0') return 0;
			break;
		}
		else if(*p!=*s) return 0;
	}
	if(*p=='/') ++p;
	return *s=='\0'?p:0;
}
/*
 * Read entries from dir, one at a time.
 * edir points at the nul at the end of dir.
 * The buffer pointed to by dir is guaranteed to be long enough to hold
 * any matching file name.
 */
void matchdir(char *dir, char *edir, char *pattern){
	DIR *fd;
	struct dirent *dp;
	struct stat st;
	int n;
	char *npattern, *s, *t;
	*edir='\0';
	fd=opendir(*dir?dir:".");
	if(fd==0) return;
	while((dp=readdir(fd))!=0){
		if(npattern=match(dp->d_name, pattern)){
			t=edir;
			if(edir!=dir && edir[-1]!='/') *t++='/';
			for(s=dp->d_name;*s;s++,t++) *t=*s;
			*t='\0';
			if(*npattern=='\0'){
				arg(chargp);
				for(s=dir;*s;s++) argchr(*s);
				argchr('\0');
			}
			else if(stat(dir, &st)==0 && S_ISDIR(st.st_mode))
				matchdir(dir, t, npattern);
		}
	}
	closedir(fd);
}
int globcmp(const void *a, const void *b){
	return strcmp(*(char **)a, *(char **)b);
}
#define	NAMELEN	256 /* file name components can be no longer than this */
void glob(void){
	char *s, *t, *buf, *pattern, **p;
	int mark;
	int nbuf;
	mark=argp-args-1;
	pattern=strdup(argp[-1]);
	nbuf=strlen(pattern+1);
	for(s=pattern;*s;s++) if(*s==GLOB && *++s=='*') nbuf+=NAMELEN;
	buf=malloc(nbuf);
	if(buf==0) nomem();
	if(pattern[0]=='/'){
		buf[0]='/';
		matchdir(buf, buf+1, pattern+1);
	}
	else
		matchdir(buf, buf, pattern);
	if(argp==args+mark+1){	/* no match, just remove GLOB markers */
		for(s=t=args[mark];*t;s++,t++){
			if(*s==GLOB) s++;
			*t=*s;
		}
	}
	else{			/* discard pattern, sort matches */
		for(p=args+mark+1;p!=argp;p++) p[-1]=p[0];
		--argp;
		p=args+mark;
		qsort(p, argp-p, sizeof(char *), globcmp);
	}
	free(pattern);
	free(buf);
}
void word(int c){
	int doglob;
	arg(chargp);
	doglob=0;
	for(;;){
		switch(c){
		case ' ':
		case '\t':
		case '&':
		case ';':
		case '\n':
		case '|':
		case '<':
		case '>':
			peekc=c;
			goto Done;
		case '\\':
			c=readc();
			if(c=='\n' || c==EOF){
				peekc=' ';
				goto Done;
			}
			argchr(c);
			break;
		case '\'':
			while((c=readc())!='\'' && c!=EOF) argchr(c);
			break;
		case '*':
		case '?':
		case GLOB:
			argchr(GLOB);
			doglob=1;
			argchr(c);
			break;
		default:
			argchr(c);
			break;
		}
		c=readc();
	}
Done:
	argchr('\0');
	if(!skip && doglob) glob();
}
void redirect(int fd){
	int c, rfd;
	int mark;
	char *op;
	do
		c=readc();
	while(c==' ' || c=='\t');
	mark=argp-args;
	word(c);
	if(!skip){
		if(redir[fd]!=fd) err("duplicate redirection\n");
		else if(argp!=args+mark+1) err("ambiguous redirection\n");
		else{
			if(fd==1){
				op="create";
				rfd=creat(argp[-1], 0666);
			}
			else{
				op="open";
				rfd=open(argp[-1], O_RDONLY);
			}
			if(rfd==-1) err("can't %s %s\n", op, argp[-1]);
			else
				redir[fd]=rfd;
		}
	}
	argp=args+mark;
	chargp=*argp;
}
void doprompt(void){
	skip=0;
	write(2, "% ", 2);
}
jmp_buf restart;
void catch(void){
	longjmp(restart, 0);
}
char *ipath[]={
	".",
	"/bin",
	"/usr/bin",
};
void main(int argc, char *argv[]){
	int c, prompt;
	shname=argv[0];
	switch(argc){
	default:
		fprintf(stderr, "Usage: %s [file]\n", shname);
		exit(1);
	case 1:
		fd=0;
		prompt=1;
		break;
	case 2:
		fd=open(argv[1], O_RDONLY);
		if(fd==-1){
			fprintf(stderr, "%s: can't open %s\n", shname, argv[1]);
			exit(1);
		}
		prompt=0;
		break;
	}
	setpath(ipath, ipath+(sizeof ipath/sizeof ipath[0]));
	args=malloc(100*sizeof(char *));
	if(args==0) nomem();
	argp=args;
	eargs=args+100;
	charg=malloc(1000*sizeof(char));
	if(charg==0) nomem();
	chargp=charg;
	echarg=charg+100;
	redir[0]=0;
	redir[1]=1;
	epid=pid;
	if(prompt){
		setjmp(restart);
		signal(SIGINT, catch);
		signal(SIGQUIT, catch);
		doprompt();
	}
	for(;;){
		c=readc();
		switch(c){
		case EOF: exit(0);
		case ' ': break;
		case '\t': break;
		case '&':
			if(nextis('&')){
				cmd(0, 0);
				if(!success) skip=1;
			}
			else
				cmd(0, 1);
			break;
		case '|':
			if(nextis('|')){
				cmd(0, 0);
				if(success) skip=1;
			}
			else
				pipeline();
			break;
		case ';': cmd(0, 0); break;
		case '#':
			do{
				c=readc();
				if(c==EOF) exit(0);
			}while(c!='\n');
		case '\n':
			cmd(0, 0);
			if(prompt) doprompt();
			break;
		case '<': redirect(0); break;
		case '>': redirect(1); break;
		default: word(c); break;
		}
	}
}
