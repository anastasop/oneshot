#include <u.h>
#include <libc.h>
#include <thread.h>
#include <mux.h>

Mux mux;
Ioproc *readProc;
Ioproc *writeProc;
int netfd;

typedef struct Message Message;
struct Message {
	uint tag;
	uint val;
};

int MessageSetTag(Mux *mux, void *msg, uint tag)
{
	return ((Message*)msg)->tag = tag;
}

int MessageGetTag(Mux *mux, void *msg)
{
	return ((Message*)msg)->tag;
}

int NetSend(Mux *mux, void *msg)
{
	uint vals[2];
	Message *m;

	m = (Message*)msg;
	vals[0] = m->tag;
	vals[1] = m->val;

	return iowrite(writeProc, netfd, vals, sizeof(vals));
}

void*
NetRecv(Mux *mux)
{
	int nr;
	uint vals[2];
	Message *msg;

	nr = ioreadn(readProc, netfd, vals, sizeof(vals));
	if (nr != sizeof(vals)) {
		fprint(2, "Expected %d Got %d", sizeof(vals), nr);
		return nil;
	}
	msg = malloc(sizeof(Message));
	msg->tag = vals[0];
	msg->val = vals[1];
	return msg;
}

#define EOFVAL 128000

void
muxfunc(void *arg)
{
	Message request, *response;

	request.val = (long)arg;
	response = muxrpc(&mux, &request);
	fprint(1, "Received [%d %d]\n", response->tag, response->val);
	free(response);
}

void
threadmain(int argc, char *argv[])
{
	int i, n;
	uint *reqVals;

	if (argc > 1) {
		n = atoi(argv[1]);
	} else {
		n = 16;
	}

	reqVals = (uint*)malloc(n * sizeof(uint));
	for (i = 0; i < n; i++) {
		reqVals[i] = rand() % 1024;
	}
	reqVals[n - 1] = EOFVAL;

	readProc = ioproc();
	writeProc = ioproc();

	mux.mintag = 0;
	mux.maxtag = 65536;
	mux.settag = MessageSetTag;
	mux.gettag = MessageGetTag;
	mux.send = NetSend;
	mux.recv = NetRecv;
	muxinit(&mux);
//	muxprocs(&mux);

	netfd = dial("tcp!localhost!40000", nil, nil, nil);
	assert(netfd != -1);

	for (i = 0; i < n; i++) {
		threadcreate(muxfunc, (void*)(long)reqVals[i], 8192);
	}
}
