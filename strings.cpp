#include <string>
#include <iostream>
using namespace std;

int
main()
{
	string s = "spy";
// nop	string s = 'a';
// nop	string s = 37;

	string::value_type c = s[0];
	string::reference c0 = s[0];
	string::reference c1 = s.at(0);

	s.length(); s.size();

	string::iterator p = s.begin(), t = s.end();
	
	cout << s << endl;

	s = "ros";
	cout << s << endl;
	s.assign("one");
	cout << s << endl;

	// note, a C++ string can contain 0 many times
	cout << string("Hello").c_str() << endl;

	cout << string("A").compare(string("B")) << endl;
	cout << (string("A") > string("B")) << endl;

	string st = "spyros";
	st += "!";
	st.append("!");
	cout << st << endl;

	cout << string("spy") + "ros" << endl;

	// find
	// replace
	// substr
}
