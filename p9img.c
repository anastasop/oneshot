#include <u.h>
#include <libc.h>
#include <draw.h>
#include <event.h>
#include <cursor.h>

#define NCOLORS 8
#define NITEMS 1024

typedef struct Item Item;
struct Item {
	Rectangle r;
	Image *color;
};

Image *colors[NCOLORS];
Item items[NITEMS];
int nitem = 0;

void
drawitem(Item it)
{
	Point points[4];

	points[0] = it.r.min;
	points[1] = Pt(it.r.max.x, it.r.min.y);
	points[2] = it.r.max;
	points[3] = Pt(it.r.min.x, it.r.max.y);

	fillpoly(screen, points, 4, 1, it.color, Pt(0, 0));
}

void
grey1(Rectangle r)
{
	Image *img;
	uchar *data;
	char *chan;
	int i, ndata, d;
	Point points[4];

	chan = "k8a8";
	d = chantodepth(strtochan(chan));

	img = allocimage(display, r, strtochan(chan), 0, DNofill);
	if (img == nil) {
		sysfatal("cannot allocate image: %r");
	}

	ndata = bytesperline(r, d) * Dy(r);
	data = (uchar*)malloc(ndata);
	if (data == nil) {
		sysfatal("no memory: %r");
	}

	for (i = 0; i < ndata; i++) {
		data[i] = nrand(256);
	}

	loadimage(img, r, data, ndata);
	free(data);

	points[0] = r.min;
	points[1] = Pt(r.max.x, r.min.y);
	points[2] = r.max;
	points[3] = Pt(r.min.x, r.max.y);

	fillpolyop(screen, points, 4, 1, img, r.min, SxorD);
}


void
eresized(int new)
{
	int i;

	if (new && getwindow(display, Refnone) < 0) {
		sysfatal("attach to window: %r");
	}

	draw(screen, screen->r, display->white, display->opaque, Pt(0, 0));

	for (i = 0; i < nitem; i++) {
		drawitem(items[i]);
	}
}

int
main(int argc, char **argv)
{
  	Mouse m;
	Rectangle r;
	Item it;

	ARGBEGIN {
	} ARGEND

	if (initdraw(0, 0, "chdb") < 0) {
		sysfatal("initdraw failed: %r");
	}

	einit(Emouse);

	colors[0] = allocimagemix(display, DRed, DRed);
	colors[1] = allocimagemix(display, DGreen, DGreen);
	colors[2] = allocimagemix(display, DBlue, DBlue);
	colors[3] = allocimagemix(display, DYellow, DYellow);
	colors[4] = allocimagemix(display, DMagenta, DMagenta);
	colors[5] = allocimagemix(display, DBlack, DBlack);
	colors[6] = allocimagemix(display, DCyan, DCyan);
	colors[7] = allocimagemix(display, DBluegreen, DBluegreen);

	print("Depth of k1 is %d\n", chantodepth(strtochan("k1")));
	print("Depth of k8 is %d\n", chantodepth(strtochan("k8")));
	print("Depth of r8g8b8 is %d\n", chantodepth(strtochan("r8g8b8")));
	print("Depth of GREY1 is %d\n", chantodepth(GREY1));
	print("Depth of GREY2 is %d\n", chantodepth(GREY2));

	r = Rect(0, 0, 64, 64);
	print("Bytes per line for rect %R and depth 1 is %d\n", r, bytesperline(r, 1));
	print("Bytes per line for rect %R and depth 8 is %d\n", r, bytesperline(r, 8));
	print("Bytes per line for rect %R and depth 24 is %d\n", r, bytesperline(r, 24));

	for(;;) {
		m = emouse();
		if (m.buttons & 4) {
			exits(nil);
		}
		if (m.buttons & 1) {
			print("Window: %R\n", screen->r);
			m.buttons = 7;
			r = egetrect(1, &m);
			print("Rectangle: %R\n", r);
			if (Dx(r) > 0 && Dy(r) > 0) {
				it.color = colors[nrand(NCOLORS)];
				it.r = r;
				items[nitem++] = it;
				// drawitem(it);
				grey1(r);
			}
		}
	}

	exits(nil);
	return 0;
}
