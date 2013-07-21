/*% date '+char version[]="%%Y.%%m.%%d-%%T-'`hostname`'";' >vers.h && cc -o # %
 * A tiny http daemon.
 *
 * The point of this program is to provide relatively fast http/1.1
 * service in a secure and small program.
 * To that end, the list of urls the program will serve
 * is compiled in, to minimize the chance that the program
 * will give away data other than you intend.
 *
 * See the comment just before #include "config.h" for a short
 * description of the configuration information.
 *
 * The only methods recognized are GET, HEAD and TRACE.
 *
 * Bugs:
 *	Doesn't look at headers, so doesn't do conditional GETs
 *	or range-restricted transfers.
 *	Doesn't do chunked transfers.
 *	Should do urlencoded POST querys.
 *	Needs to hang up persistent connections after a time out.
 *	Depending on inetd to start us is convenient, but I
 *	think it limits the rate at which we can process connections.
 *	I should write a program to generate the resource table
 *	from a more tractable representation.
 *	When misconfigured, this will occasionally send a local
 *	pathname to the client, for example, when runget is asked
 * 	to run a command on which the shell chokes.
 */
#include <stdio.h>
#include <time.h>
#include <stdarg.h>
#include <stdlib.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <errno.h>
#include "vers.h"
typedef struct Message Message;
typedef struct Header Header;
typedef struct Resource Resource;
typedef struct Fn Fn;
struct Message{
	int method;
	char *methodname;
	char *uri;
	Resource *resource;
	char *query;
	int version[2];
	int nheader;
	Header *header;
	char *text;
};
struct Header{
	char *name;
	char *value;
};
struct Resource{
	char *name;		/* A uri that might appear in a request */
	char *type;		/* Content-type of the resource */
	Fn *fn;			/* len/get functions */
	char *arg[2];		/* arguments used by functions */
};
struct Fn{
	int (*len)(Message *);	/* Returns the length of the resource */
	void (*get)(Message *);	/* sends the body of the resource */
};
#define	NOTFOUND	(-2)	/* len returns this if no file */
#define	NOLEN		(-1)	/* len returns this if length not known */
int filelen(Message *);
void fileget(Message *);
int runlen(Message *);
void runget(Message *);
Fn run[1]={runlen, runget};
Fn file[1]={filelen, fileget};
char html[]="text/html";
char plain[]="text/plain";
char gif[]="image/gif";
void *emalloc(int);
void serverabort(int, char *);
/*
 * Configuration information is kept in config.h,
 * which must contain initializers for the following variables:
 *
 * char root[];		// Prepended to relative path names.
 * char logname[];	// The name of the log file.
 * char getlogname[];	// A separate log listing urls retrieved
 * char getloglock[];	// An unused name in the same directory as getlogname,
 *			// used for locking.
 * char *runenv[];	// the environment to be used when executing scripts
 * Resource resource[];	// The list of resources and service methods.
 *
 * Here's an example that returns /usr/spool/http/index.html
 * when asked for / or /index.html, returns /etc/passwd when
 * asked for /security.breach, and runs the program /usr/spool/http/bin/search
 * when asked for /cgi-bin/search?<anything> and returns.
 *
 * char root[]="/usr/spool/http";
 * char logname[]="/usr/spool/http/log";
 * char getlogname[]="/usr/spool/http/namelog";
 * char getloglock[]="/usr/spool/http/namelock";
 * char *runenv[]={
 *	"ROOT=/usr/spool/http",
 *	"PATH=:/bin:/usr/bin:/usr/spool/http/bin",
 *	0
 * };
 * Resource resource[]={
 *	"",                 html,  file, "index.html",  0,
 *	"index.html",       html,  file, "index.html",  0,
 *	"security.breach",  plain, file, "/etc/passwd", 0,
 *	"cgi-bin/search?",  html,  run,  "bin/search",  0,
 *	0
 * };
 *
 * Note that url names (the first field of a Resource) should be in
 * canonical form as determined by pathcanon, below.  That is, they
 * should have neither leading nor trailing nor pairs of adjacent
 * slashes, and no component should be . or .., except .. may be
 * the first component.
 */
#include "config.h"
char white[]=" \t";
#define	OPTIONS	0
#define	GET	1
#define	HEAD	2
#define	POST	3
#define	PUT	4
#define	DELETE	5
#define	TRACE	6
#define	CONNECT	7
#define	UNKNOWN	8
char *method[]={
	"OPTIONS",
	"GET",
	"HEAD",
	"POST",
	"PUT",
	"DELETE",
	"TRACE",
	"CONNECT",
	0
};
FILE *logfd;
char *weekday[7]={ "Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat" };
char *month[12]={
	"Jan", "Feb", "Mar", "Apr", "May", "Jun",
	"Jul", "Aug", "Sep", "Oct", "Nov", "Dec"
};
char server[]="tdhttp";
/*
 * Add an entry to the log.
 * Not locked (to avoid serialization on the lock),
 * so messages can be scrambled.
 */
void log(char *fmt, ...){
	va_list args;

	va_start(args, fmt);
	vfprintf(logfd, fmt, args);
	va_end(args);
	fflush(logfd);
}
void logwrite(void *buf, int nbuf){
	fwrite(buf, nbuf, 1, logfd);
	fflush(logfd);
}
/*
 * There's a separate log just listing
 * URLs on which a GET was done, just
 * so that you can get page statistics
 * by running
 *	sort getlog|uniq -c
 * Locked to avoid corruption.
 * It worries me that the lock might cause serialization.
 */
#define	NLOCKLOOP	10
void getlog(char *url){
	FILE *fd;
	int i, f0, f1, f2;

	f0=1;
	f1=1;
	for(i=0;i!=NLOCKLOOP && link(getlogname, getloglock)==-1;i++){
		if(errno==EEXIST){
			/* link exists -- someone's holding the lock */
			if(f0) sleep(f0);
			f2=f0+f1;
			f0=f1;
			f1=f2;
		}
		else{
			log("bad getlog lock: %s\n", strerror(errno));
			break;
		}
	}
	if(i==NLOCKLOOP)
		log("Breaking stale getlog lock (waited %d sec)\n", f2-1);
	fd=fopen(getlogname, "a");
	fprintf(fd, "/%s\n", url);
	fclose(fd);
	unlink(getloglock);
}
/*
 * Convert a UNIX time to an RFC 822/1123 date
 * Buffer must be 30 bytes.
 */
char *fmtdate(char *buf, time_t t){
	struct tm *tm;

	tm=gmtime(&t);
	sprintf(buf, "%s, %02d %s %d %02d:%02d:%02d GMT",
		weekday[tm->tm_wday], tm->tm_mday, month[tm->tm_mon],
		tm->tm_year+1900, tm->tm_hour, tm->tm_min, tm->tm_sec);
	return buf;
}
/*
 * Convert status codes into prose
 */
char *statusmsg(int code){
	switch(code){
	default:  return "GOK";
	case 100: return "Continue";
	case 101: return "Switching protocols";
	case 200: return "OK";
	case 201: return "Created";
	case 202: return "Accepted";
	case 203: return "Non-Authoritative Information";
	case 204: return "No Content";
	case 205: return "Reset Content";
	case 206: return "Partial Content";
	case 207: return "Partial Update OK";
	case 300: return "Multiple Choices";
	case 301: return "Moved Permanently";
	case 302: return "Moved Temporarily";
	case 303: return "See Other";
	case 304: return "Not Modified";
	case 305: return "Use Proxy";
	case 307: return "Temporary Redirect";
	case 400: return "Bad Request";
	case 401: return "Unauthorized";
	case 402: return "Payment Required";
	case 403: return "Forbidden";
	case 404: return "Not Found";
	case 405: return "Method Not Allowed";
	case 406: return "Not Acceptable";
	case 407: return "Proxy Authentication Required";
	case 408: return "Request Timeout";
	case 409: return "Conflict";
	case 410: return "Gone";
	case 411: return "Length Required";
	case 412: return "Precondition Failed";
	case 413: return "Request Entity Too Large";
	case 414: return "Request-URI Too Long";
	case 415: return "Unsupported Media Type";
	case 416: return "Requested range not satisfiable";
	case 417: return "Expectation Failed";
	case 418: return "Reauthentication Required";
	case 419: return "Proxy Reauthentication Required";
	case 500: return "Internal Server Error";
	case 501: return "Not Implemented";
	case 502: return "Bad Gateway";
	case 503: return "Service Unavailable";
	case 504: return "Gateway Timeout";
	case 505: return "HTTP Version Not Supported";
	case 506: return "Partial Update Not Implemented";
	}
}
/*
 * Send the response message and headers that begin every response.
 */
void reply(int code, char *type, int len, Header *h, int nh){
	char dbuf[30];
	int i;

	printf("HTTP/1.1 %d %s\r\n", code, statusmsg(code));
	printf("Date: %s\r\n", fmtdate(dbuf, time(0)));
	printf("Server: %s/%s\r\n", server, version);
	if(type) printf("Content-type: %s\r\n", type);
	if(len!=NOLEN) printf("Content-length: %d\r\n", len);
	for(i=0;i!=nh;i++) printf("%s: %s\r\n", h[i].name, h[i].value);
	printf("\r\n");
}
void htmlstr(int code, char *str){
	reply(code, html, strlen(str), 0, 0);
	printf("%s", str);
}
void htmlmsg(int code, char *msg){
	char buf[8192];

	sprintf(buf, "<head><title>%s server message</title></head\n"
		     "<body><h3>Message from %s</h3>\n"
		     "%s</body>\n",
			server, server, (char *)msg);
	htmlstr(code, buf);
}
void serverabort(int code, char *msg){
	char buf[4096];

	log("%s/%s server abort %s at %s\n", server, version, msg,
		fmtdate(buf, time(0)));
	sprintf(buf, "Server aborted: %s", msg);
	htmlmsg(code, buf);
	exit(1);
}
void servererror(int code, char *msg){
	char buf[4096];

	log("%s/%s server error %d %s\n", server, version, code, msg);
	sprintf(buf, "Server error %d: %s", code, msg);
	htmlmsg(code, buf);
}
void clienterror(int code, char *msg){
	char buf[4096];

	log("%s/%s client error %d %s\n", server, version, code, msg);
	sprintf(buf, "Client error %d: %s", code, msg);
	htmlmsg(code, buf);
}
void *emalloc(int n){
	void *buf;

	buf=malloc(n);
	if(buf==0) serverabort(500, "out of memory");
	return buf;
}
void *erealloc(void *buf, int n){
	buf=realloc(buf, n);
	if(buf==0) serverabort(500, "out of memory");
	return buf;
}
/*
 * Return a pointer to the first character of p not in chars.
 */
char *span(char *p, char *chars){
	while(*p && strchr(chars, *p)) p++;
	return p;
}
/*
 * Return a pointer to the first character of p in chars.
 */
char *brk(char *p, char *chars){
	while(*p && !strchr(chars, *p)) p++;
	return p;
}
int lower(int c){
	return 'A'<=c && c<='Z'?c+'a'-'A':c;
}
int upper(int c){
	return 'a'<=c && c<='z'?c+'A'-'a':c;
}
/*
 * create a nul-terminated copy of a string, given a pointer and a length
 */
char *strndup(char *s, int n){
	char *v;

	v=emalloc(n+1);
	memcpy(v, s, n);
	v[n]='\0';
	return v;
}
/*
 * Case-insensitive string equality
 */
int match(char *name, char *buf){
	while(*buf) if(lower(*name++)!=lower(*buf++)) return 0;
	return *name=='\0';
}
/*
 * Capitalize a field name, in place.
 */
void capitalize(char *cp){
	if(*cp=='\0') return;
	*cp=upper(*cp);
	while(*++cp) *cp=cp[-1]=='-'?upper(*cp):lower(*cp);
}
/*
 * Read a client message and its header lines.
 * Returns a pointer to a mallocked list of pointers to the beginnings
 * of lines, ending with a pointer past the end of the last line and a 0
 * pointer.
 *
 * Just reads until it sees two newlines, possibly with interspersed crs,
 * in a row.
 */
#define	MINCR	300
#define	LINCR	10
char **rdmsg(void){
	char *mbuf, *mp, *embuf;
	char **lbuf, **lp, **elbuf;
	int nnl, nmbuf, nlbuf, c;

#define	savec(chr)\
	if(mp==embuf){\
		nmbuf+=MINCR;\
		mbuf=erealloc(mbuf, nmbuf);\
		embuf=mbuf+nmbuf;\
		mp=embuf-MINCR;\
		*mp++=(chr);\
	}\
	else *mp++=(chr)
#define	saveline(line)\
	if(lp==elbuf){\
		nlbuf+=LINCR;\
		lbuf=erealloc(lbuf, nlbuf*sizeof(char *));\
		elbuf=lbuf+nlbuf;\
		lp=elbuf-LINCR;\
		*lp++=(line);\
	}\
	else *lp++=(line)

	nmbuf=400;
	mbuf=emalloc(nmbuf);
	embuf=mbuf+nmbuf;
	mp=mbuf;
	nlbuf=20;
	lbuf=emalloc(nlbuf*sizeof(char *));
	elbuf=lbuf+nlbuf;
	lp=lbuf;
	nnl=1;
	saveline(mbuf);
	do{
		switch(c=getchar()){
		default:
			savec(c);
			nnl=0;
			break;
		case EOF:
			if(mp!=mbuf)
				clienterror(400,
					"EOF detected while reading request");
			free(mbuf);
			free(lbuf);
			return 0;
		case '\r':
			savec(c);
			break;
		case '\n':
			nnl++;
			savec(c);
			saveline(mp);
			break;
		}
	}while(nnl!=2);
	logwrite(mbuf, mp-mbuf);
	saveline(0);
	return lbuf;
}
/*
 * Echo a TRACE request back to the client.
 * Returns 1 or 0 depending on whether or not the message was a TRACE.
 * Parsing is extremely charitable -- the only requirement is that
 * the first word in the message be TRACE, in upper or lower case.
 */
int dotrace(char **msg){
	char *mp;
	char **emsg;

	for(mp=msg[0];mp!=msg[1] && *mp==' ' || *mp=='\t';mp++);
	if(msg[1]-mp<6
	|| lower(mp[0])!='t'
	|| lower(mp[1])!='r'
	|| lower(mp[2])!='a'
	|| lower(mp[3])!='c'
	|| lower(mp[4])!='e'
	|| !strchr(" \t\r\n", mp[5]))
		return 0;
	for(emsg=msg;emsg[1];emsg++);
	reply(200, "message/http", *emsg-*msg, 0, 0);
	fwrite(*msg, *emsg-*msg, 1, stdout);
	return 1;
}
/*
 * Convert lines into nul-terminated strings,
 * deleting \r and \n and merging continuation lines.
 * This rewrites the strings (which start at mbuf[0])
 * and the pointers to them (which start at mbuf).
 *
 * Note that this returns -1 if the first line is empty,
 * indicating that the request line was missing.
 */
int continuation(char **mbuf){
	char **dlp, **slp, *dcp, *cp;
	int n;

	dlp=mbuf;	/* destination line pointer */
	dcp=mbuf[0];	/* destination character pointer */
	for(slp=mbuf;slp[1];slp++){
		/*
		 * slide the line to the left
		 */
		n=slp[1]-slp[0];
		memmove(dcp, slp[0], n);
		/*
		 * Advance dcp to the end of the line,
		 * forgetting the terminating \r, \n or \r\n
		 */
		dcp+=n-1;
		if(*dcp=='\r') --dcp;
		/*
		 * If there's no continuation line, put a nul on the end
		 * and record the start of the (provisional) next line.
		 *
		 * The first line can't have a continuation.  Other lines
		 * don't have a continuation if there's no line following
		 * or the following line starts with a non-blank, non-tab.
		 */
		if(slp==mbuf || slp[2]==0 || *slp[1]!=' ' && *slp[1]!='\t'){
			*dcp++='\0';
			*++dlp=dcp;
		}
	}
	dlp[-1]=0;	/* forget the blank line at the end */
	return dlp-mbuf-2;
}
/*
 * Break fields out of a request line, which should look like
 *	GET /a/b/c/d HTTP/1.1
 */
void parserequest(Message *m, char *cp){
	int i;
	char *query;

	m->methodname=span(cp, white);
	cp=brk(m->methodname, white);
	*cp++='\0';
	for(i=0;method[i];i++) if(match(method[i], m->methodname)) break;
	m->method=i;
	m->uri=span(cp, white);
	cp=brk(m->uri, white);
	*cp++='\0';
	query=strchr(m->uri, '?');
	if(query){
		m->query=strdup(query+1);
		query[1]='\0';
	}
	else
		m->query=0;
	cp=span(cp, white);
	if(lower(cp[0])=='h'
	&& lower(cp[1])=='t'
	&& lower(cp[2])=='t'
	&& lower(cp[3])=='p'
	&& cp[4]=='/'){
		cp=span(cp+5, white);
		m->version[0]=atoi(cp);
		cp=span(brk(cp, "."), ".");
		m->version[1]=atoi(cp);
	}
	else{
		m->version[0]=1;
		m->version[1]=1;
	}
}
/*
 * Break fields out of a header line, which should look like
 *	Name: value
 */
void parseheader(Header *h, char *cp){
	h->name=span(cp, white);
	cp=brk(h->name, " \t:");
	h->value=span(cp, white);
	if(*h->value==':') h->value=span(cp+1, white);
	*cp='\0';
	capitalize(h->name);
}
/*
 * Read a client message and build a message structure for it.
 * This short circuits TRACE requests, because the code breaks
 * the request buffer into tokens in place and TRACE requires that
 * we return it verbatim.
 */
Message *getclient(void){
	char **mbuf;
	Message *m;
	int i, nheader;

Again:
	mbuf=rdmsg();
	if(mbuf==0) return 0;
	if(dotrace(mbuf)){
		free(mbuf[0]);
		free(mbuf);
		goto Again;
	}
	nheader=continuation(mbuf);
	if(nheader<0){
		clienterror(400, "Message from client contains no request");
		free(mbuf[0]);
		free(mbuf);
		return 0;
	}
	m=emalloc(sizeof(Message));
	m->nheader=nheader;
	m->header=emalloc(m->nheader*sizeof(Header));
	m->text=mbuf[0];
	parserequest(m, mbuf[0]);
	for(i=0;i!=nheader;i++) parseheader(&m->header[i], mbuf[i+1]);
	free(mbuf);
	return m;
}
void freemessage(Message *m){
	Header *h, *next;

	free(m->text);
	free(m->header);
	free(m);
}
/*
 * Reduce a file name in place to a canonical form containing
 * no empty component, no component "." and no non-initial component "..".
 */
void pathcanon(char *file){
	char *s, *start, *t, *nt;

	s=file;
	start=file;
	/*
	 * Advance t past one component per iteration of the loop.
	 * First identify the beginning of the next component
	 * and arrange that the current component ends in a nul.
	 * If the component is .. and there is a previous non-..
	 * component, discard it and the previous component.
 	 * Otherwise if the component is neither empty nor '.',
	 * we copy it from t to s, advancing both.
	 */
	for(t=file;*t!='\0';t=nt){
		for(nt=t;*nt!='/' && *nt!='\0';nt++);
		if(*nt=='/') *nt++='\0';
		if(s!=start && strcmp(t, "..")==0){
			do
				--s;
			while(s!=start && s[-1]!='/');
		}
		else if(strcmp(t, "")!=0 && strcmp(t, ".")!=0){
			/*
			 * First arrange that leading .. cannot
			 * be deleted by a subsequent .. by advancing
			 * start past ../, then copy the component.
			 */
			if(strcmp(t, "..")==0) start=s+3;
			while(*t!='\0') *s++=*t++;
			*s++='/';
		}
	}
	/*
	 * If the loop copied any components, it put a slash on the end
	 * which we now must delete or else there may not be room
	 * for the nul.
	 */
	if(s!=file) --s;
	*s='\0';
}
/*
 * Return value is 0 if the connection must be closed because
 * the length of the reply could not be determined and must
 * be inferred by the client from EOF on the connection, and 1 otherwise.
 */
int sendreply(Message *m){
	char *buf;
	Resource *rp;
	int len;

	/*
	 * Should be a binary search.
	 */
	pathcanon(m->uri);
	for(rp=resource;rp->name;rp++)
		if(strcmp(rp->name, m->uri)==0) break;
	if(rp->name==0){
	NotFound:
		buf=emalloc(strlen(m->uri)+30);
		sprintf(buf, "%s not found", m->uri);
		clienterror(404, buf);
		free(buf);
		return 1;
	}
	getlog(m->uri);
	m->resource=rp;
	switch(m->method){
	default:
		serverabort(500, "Missing case in sendreply");
	case TRACE:
		serverabort(500, "TRACE unexpected in sendreply");
	case GET:
		len=rp->fn->len(m);
		if(len==NOTFOUND) goto NotFound;
		reply(200, rp->type, len, 0, 0);
		rp->fn->get(m);
		return len!=NOLEN;
	case HEAD:
		len=rp->fn->len(m);
		if(len==NOTFOUND) goto NotFound;
		reply(200, rp->type, len, 0, 0);
		return 1;
	case POST:
		servererror(501, "POST not implemented");
		return 1;
	case OPTIONS:
		servererror(501, "OPTIONS not implemented");
		return 1;
	case PUT:
		servererror(501, "PUT not implemented");
		return 1;
	case DELETE:
		servererror(501, "DELETE not implemented");
		return 1;
	case CONNECT:
		servererror(501, "CONNECT not implemented");
		return 1;
	case UNKNOWN:
		buf=emalloc(strlen(m->methodname)+30);
		sprintf(buf, "Unknown command %s", m->methodname);
		clienterror(400, buf);
		free(buf);
		return 1;
	}
}
char *abspath(char *rel){
	char *abs;

	if(rel[0]=='/') return strdup(rel);
	abs=emalloc(strlen(root)+strlen(rel)+2);
	sprintf(abs, "%s/%s", root, rel);
	return abs;
}
/*
 * filelen/fileget serve up plain files stored on disk.
 */
int filelen(Message *m){
	char *fullname;
	struct stat buf;
	int len;

	fullname=abspath(m->resource->arg[0]);
	len=stat(fullname, &buf)<0?NOTFOUND:buf.st_size;
	free(fullname);
	return len;
}
void fileget(Message *m){
	char *fullname;
	int fd, n;
	char buf[8192];

	fullname=abspath(m->resource->arg[0]);
	fd=open(fullname, 0);
	free(fullname);
	if(fd<0) serverabort(500, "file disappeared");
	for(;;){
		n=read(fd, buf, 8192);
		if(n<=0) break;
		fwrite(buf, n, 1, stdout);
	}
	close(fd);
}
/*
 * runlen/runget runs an arbitrary program to generate the file
 */
int runlen(Message *m){
	return NOLEN;
}
void runget(Message *m){
	char *command;
	int stat, i;
	char *argv[4];

	command=abspath(m->resource->arg[0]);
	i=0;
	argv[i++]=m->resource->arg[0];
	if(m->resource->arg[1]) argv[i++]=abspath(m->resource->arg[1]);
	if(m->query) argv[i++]=m->query;
	argv[i]=0;
	fflush(logfd);
	fflush(stdout);
	switch(fork()){
	case -1:
		serverabort(500, "can't fork");
	case 0:
		execve(command, argv, runenv);
		servererror(501, "can't exec command");
		exit(1);
	default:
		wait(&stat);
	}
	if(m->resource->arg[1]) free(argv[1]);
}
main(int argc, char *argv[], char *envp[]){
	Message *m;
	char dbuf[30];
	struct sockaddr_in peer;
	int npeer;

	logfd=fopen(logname, "a");
	log("start %s\n", fmtdate(dbuf, time(0)));
	log("server %s/%s\n", server, version);
	npeer=sizeof peer;
	peer.sin_addr.s_addr=~0;	/* in case of testing */
	getpeername(0, &peer, &npeer);
	log("client %s\n", inet_ntoa(peer.sin_addr));
	while((m=getclient())!=0 && sendreply(m)){
		fflush(stdout);
		if(m->version[0]<1
		|| m->version[0]==1 && m->version[1]<1)
			break;
		freemessage(m);
	}
	log("done %s\n\n", fmtdate(dbuf, time(0)));
	exit(0);
}
