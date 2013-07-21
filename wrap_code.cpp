
/*
	Program used to test basic performance of wrappers as described in
	B. Stroustrup: Wrapping C++ Member Function Calls"
	The C++ Report" June 2000.

*/


// First a measurement harness provided by Andrew Koenig and
// described in a series of JOOP columns in 1999/2000

// it repeats timing loops until it is (statistically) convinced
// that a result is meaningful


#include <climits>
#include <iostream>
#include <vector>
#include <assert.h>
#include <algorithm>
#include <string>
#include <ctime>
using namespace std;

// How many clock units does it take to interrogate the clock?
static double clock_overhead()
{
    clock_t k = clock(), start, limit;

    // Wait for the clock to tick
    do start = clock();
    while (start == k);

    // interrogate the clock until it has advanced at least a second
    // (for reasonable accuracy)
    limit = start + CLOCKS_PER_SEC;

    unsigned long r = 0;
    while ((k = clock()) < limit)
	++r;

    return double(k - start) / r;
}

// We'd like the odds to be factor:1 that the result is
// within percent% of the median
const int factor = 10;
const int percent = 20;

// Measure a function (object) factor*2 times,
// appending the measurements to the second argument
template<class F>
void measure_aux(F f, vector<double>& mv)
{
    static double ovhd = clock_overhead();

    // Ensure we don't reallocate in mid-measurement
    mv.reserve(mv.size() + factor*2);

    // Wait for the clock to tick
    clock_t k = clock();
    clock_t start;

    do start = clock();
    while (start == k);

    // Do 2*factor measurements
    for (int i = 2*factor; i; --i) {
	unsigned long count = 0, limit = 1, tcount = 0;
	const clock_t clocklimit = start + CLOCKS_PER_SEC/100;
	clock_t t;

	do {
	    while (count < limit) {
		f();
		++count;
	    }
	    limit *= 2;
	    ++tcount;
	} while ((t = clock()) < clocklimit);

	// Wait for the clock to tick again;
	clock_t t2;
	do ++tcount;
	while ((t2 = clock()) == t);

	// Append the measurement to the vector
	mv.push_back(((t2 - start) - (tcount * ovhd)) / count);

	// Establish a new starting point
	start = t2;
    }
}

// Returns the number of clock units per iteration
// With odds of factor:1, the measurement is within percent% of
// the value returned, which is also the median of all measurements.
template<class F>
double measure(F f)
{
    vector<double> mv;

    int n = 0;			// iteration counter
    do {
	++n;

	// Try 2*factor measurements
	measure_aux(f, mv);
	assert(mv.size() == 2*n*factor);

	// Compute the median.  We know the size is even, so we cheat.
	sort(mv.begin(), mv.end());
	double median = (mv[n*factor] + mv[n*factor-1])/2;

	// If the extrema are within threshold of the median, we're done
	if (mv[n] > (median * (100-percent))/100 &&
	    mv[mv.size() - n - 1] < (median * (100+percent))/100)
	    return median;

    } while (mv.size() < factor * 200);

    // Give up!
    clog << "Help!\n\n";
    exit(1);
}


// ----------------------------------

/*
	Here come the wrapper classes,

	SWrap is the simple wrapper described in the "Prefix and Suffix"
	section using functions as prefix/suffix

	SOWrap is the simple wrapper described in the "Prefix and Suffix"
	section using function objects as prefix/suffix

	Wrap is the "robust" wrapper described in the "Parameterization"
	section.

*/

inline void prefix() { /* cout << "prefix "; */ }
inline void suffix() { /* cout << " suffix\n";  */}

struct Pref { void operator()() const { /* cout << "Pre ";*/ } };
struct Suf { void operator()() const {  /* cout << " Suf\n"; */} };


// simple/naive wrapper (fct):

template<class T>
class SCall_proxy {
	T* p;
public:
	SCall_proxy(T* pp) :p(pp){ }
	~SCall_proxy() { suffix(); }
	T* operator->() { return p; }
};

template<class T>
class SWrap {
	T* p;
public:
	SWrap(T* pp) :p(pp) { } 
	SCall_proxy<T> operator->() { prefix(); return SCall_proxy<T>(p); }
};

// simple/naive wrapper (obj):

Pref pr;
Suf su;

template<class T>
class SOCall_proxy {
	T* p;
public:
	SOCall_proxy(T* pp) :p(pp){ }
	~SOCall_proxy() { su(); }
	T* operator->() { return p; }
};

template<class T>
class SOWrap {
	T* p;
public:
	SOWrap(T* pp) :p(pp) { } 
	SOCall_proxy<T> operator->() { pr(); return SOCall_proxy<T>(p); }
};


// robust wrapper:

template<class T, class Pref, class Suf> class Wrap;

// for MS C++, remove the friend declaration and make all members of
// Call_proxy public (MS C++ 6.0 doesn't support templates as friends)

template<class T, class Suf>
class Call_proxy {
	T* p;
	mutable bool own;
	Suf suffix;

	Call_proxy(T* pp, Suf su) :p(pp), own(true), suffix(su) { }		// restrict creation

	Call_proxy& operator=(const Call_proxy&);	// prevent assignment
public:
	template<class U, class P, class S> friend class Wrap;

	Call_proxy(const Call_proxy& a)
		: p(a.p), own(true), suffix(a.suffix) { a.own=false; }

	~Call_proxy() { if (own) suffix(); }

	T* operator->() const  { return p; }
};

template<class T, class Pref, class Suf>
class Wrap {
	T* p;
	int* owned;
	void incr_owned() { if (owned) ++*owned; }
	void decr_owned() { if (owned && --*owned == 0) { delete p; delete owned; } }

	Pref prefix;
	Suf suffix;
public:
	Wrap(T& x, Pref pr, Suf su)
		:p(&x), owned(0), prefix(pr), suffix(su) { } 

	Wrap(T* pp, Pref pr, Suf su)
		:p(pp), owned(new int(1)), prefix(pr), suffix(su) { } 

	Wrap(const Wrap& a)
		:p(a.p), owned(a.owned), prefix(a.prefix), suffix(a.suffix)
		{ incr_owned(); }

	Wrap& operator=(const Wrap& a)
	{
		a.incr_owned();
		decr_owned();
		p = a.p;
		owned = a.owned;
		prefix = a.prefix;;
		suffix = a.suffix;
	}

	~Wrap() { decr_owned(); }

	Call_proxy<T,Suf> operator->() const
		{ prefix(); return Call_proxy<T,Suf>(p,suffix); }

	T* direct() const { return p; } // extract pointer to wrapped object

};

#include<iostream>
using namespace std;

class X {	// one user class
public:
	void g() const;
};

void X::g() const {/*  cout << "g()"; */}


// functions to be called by measurement():

void plain_loop_obj()
{
	X x;
	x.g();
}

void plain_loop_ptr()
{
	X x;
	X* p = &x;
	p->g();
}

void naive_wrap_obj()
{
	X x;
	SWrap<X> xx(&x);
	xx->g();
}

void naive_wrap_fct()
{
	X x;
	SWrap<X> xx(&x);
	xx->g();
}

void robust_wrap_obj()
{
	X x;
	Wrap<X,Pref,Suf> xx(x,Pref(),Suf());
	xx->g();
}

void robust_wrap_fct()
{
	X x;
	Wrap<X,void(*)(),void(*)()> xx(x,prefix,suffix);
	xx->g();
}


int main()
{
	clock_t t0 = clock();
	if (t0 == clock_t(-1)) {
		cerr << "sorry, no clock\n";
		exit(1);
	}

	double scale = 1000000/CLOCKS_PER_SEC;
	cout <<"clock ganularity: " << CLOCKS_PER_SEC << " clock ticks per second\n"
		<< "unit of output: " << scale << " microseconds per invocation\n\n";

	cout  << "plain_loop_obj() " << measure(plain_loop_obj)<< "\n"
		<< "plain_loop_ptr() " << measure(plain_loop_ptr)<< "\n"
		<< "naive_wrap_fct() " << measure(naive_wrap_fct)<< "\n"
		<< "naive_wrap_obj() " << measure(naive_wrap_obj)<< "\n"
		<< "robust_wrap_fct() " << measure(robust_wrap_fct)<< "\n"
		<< "robust_wrap_obj() " << measure(robust_wrap_obj)<< "\n";

	return 0;
}
