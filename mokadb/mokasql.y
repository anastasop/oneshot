%{
#include <iostream>
#include <string>
#include <vector>
#include <map>

extern int yyparse();
int yylex();
void yyerror(char*);

class Value;
class Node;

class Value {
public:
	std::string id;
	std::string s;
	long i;
	bool b;
};

enum NodeType {
	VALUE_NODE,
	SELECT_STMT,
	INSERT_STMT,
	UPDATE_STMT,
	DELETE_STMT,
	ORDER_STMT,
	ADD_OP,
	SUB_OP,
	MUL_OP,
	DIV_OP,
	MOD_OP,
	NEG_OP,
	AND_OP,
	OR_OP,
	NOT_OP,
	GE_OP,
	GT_OP,
	LE_OP,
	LT_OP,
	EQ_OP,
	NE_OP,
	LIKE_OP,
	NOT_LIKE_OP,
	BETWEEN_OP,
	NOT_BETWEEN_OP,
	REGEXP_OP,
	NOT_REGEXP_OP,
	IN_OP,
	NOT_IN_OP,
	ISNULL_OP,
	NOT_ISNULL_OP
};
	

class Node {
public:
	NodeType type;
	Value val;
	std::vector<Node*> args;
	std::vector<Node*> values;

	Node(NodeType t) : type(t) {}

	Node(NodeType t, Node *n1) : type(t) {
		args.push_back(n1);
	}

	Node(NodeType t, Node *n1, Node *n2) : type(t) {
		args.push_back(n1);
		args.push_back(n2);
	}

	Node(NodeType t, Node *n1, Node *n2, Node *n3) : type(t) {
		args.push_back(n1);
		args.push_back(n2);
		args.push_back(n3);
	}
};

%}

%union {
	Node* node;
	int tok;
}

%token <tok> SELECT INSERT UPDATE DELETE FROM WHERE ORDER BY VALUES ASC DESC INTO IN LIKE IS BETWEEN REGEXP
%token <node> STRING INTEGER NULLVAL ID

%left <tok> ','
%left <tok> AND OR
%right <tok> NOT
%left <tok> GT GE LT LE EQ NE
%left <tok> '+' '-'
%left <tok> '*' '/' '%'
%left <tok> UMINUS

%type <node> select_statement insert_statement delete_statement
%type <node> where_expression_opt boolean_expression condition
%type <node> value_expression value_expression_list value values
%type <node> parenthesised_list

%start command

%%
command:
	select_statement
|
	insert_statement
|
	delete_statement


select_statement:
	SELECT '*' FROM ID where_expression_opt order_by_opt
		{ $$ = new Node(SELECT_STMT, $4, $5); }


insert_statement:
	INSERT INTO ID values
		{ $$ = new Node(INSERT_STMT, $3, $4); }


delete_statement:
	DELETE FROM ID where_expression_opt
		{ $$ = new Node(DELETE_STMT, $3, $4); }


where_expression_opt:
		{ $$ = static_cast<Node*>(0); }
|
	WHERE boolean_expression
		{ $$ = $2; }


boolean_expression:
	condition
		{ $$ = $1; }
|
	boolean_expression AND boolean_expression
		{ $$ = new Node(AND_OP, $1, $3); }
|
	boolean_expression OR boolean_expression
		{ $$ = new Node(OR_OP, $1, $3); }
|
	NOT boolean_expression
		{ $$ = new Node(NOT_OP, $2); }
|
	'(' boolean_expression ')'
		{ $$ = $2; }


condition:
	value_expression '=' value_expression
		{ $$ = new Node(EQ_OP, $1, $3); }
|
	value_expression '<' value_expression
		{ $$ = new Node(LT_OP, $1, $3); }
|
	value_expression '>' value_expression
		{ $$ = new Node(GT_OP, $1, $3); }
|
	value_expression LE value_expression
		{ $$ = new Node(LE_OP, $1, $3); }
|
	value_expression GE value_expression
		{ $$ = new Node(GE_OP, $1, $3); }
|
	value_expression NE value_expression
		{ $$ = new Node(NE_OP, $1, $3); }
|
	value_expression IN parenthesised_list
		{ $$ = new Node(IN_OP, $1, $3); }
|
	value_expression NOT IN parenthesised_list
		{ $$ = new Node(NOT_IN_OP, $1, $4); }
|
	value_expression LIKE value_expression
		{ $$ = new Node(LIKE_OP, $1, $3); }
|
	value_expression NOT LIKE value_expression
		{ $$ = new Node(NOT_LIKE_OP, $1, $4); }
|
	value_expression REGEXP value_expression
		{ $$ = new Node(REGEXP_OP, $1, $3); }
|
	value_expression NOT REGEXP value_expression
		{ $$ = new Node(NOT_REGEXP_OP, $1, $4); }
|
	value_expression BETWEEN value_expression AND value_expression
		{ $$ = new Node(BETWEEN_OP, $1, $3, $5); }
|
	value_expression NOT BETWEEN value_expression AND value_expression
		{ $$ = new Node(NOT_BETWEEN_OP, $1, $4, $6); }
|
	value_expression IS NULLVAL
		{ $$ = new Node(ISNULL_OP, $1); }
|
	value_expression IS NOT NULLVAL
		{ $$ = new Node(NOT_ISNULL_OP, $1); }


values:
	VALUES parenthesised_list
		{ $$ = $2; }


parenthesised_list:
	'(' value_expression_list ')'
		{ $$ = $2; }


value_expression_list:
	value_expression
		{ $$ = $1; }
|
	value_expression_list ',' value_expression
		{ ($1)->values.push_back($3); }


value_expression:
	value
		{ $$ = $1; }
|
	ID /* column name */
		{ $$ = $1; }
|
	value_expression '+' value_expression
		{ $$ = new Node(ADD_OP, $1, $3); }
|
	value_expression '-' value_expression
		{ $$ = new Node(SUB_OP, $1, $3); }
|
	value_expression '*' value_expression
		{ $$ = new Node(MUL_OP, $1, $3); }
|
	value_expression '/' value_expression
		{ $$ = new Node(DIV_OP, $1, $3); }
|
	value_expression '%' value_expression
		{ $$ = new Node(MOD_OP, $1, $3); }
|
	'-' value_expression %prec UMINUS
		{ $$ = new Node(NEG_OP, $2); }


value:
	STRING
|
	INTEGER
|
	NULLVAL


order_by_opt:
|
	ORDER BY order_list


order_list:
	order
|
	order_list ',' order


order:
	value_expression asc_desc_opt


asc_desc_opt:
|
	ASC
|
	DESC

%%
	/* end of grammar */

void
yyerror(char *str)
{
	std::cerr << str << std::endl;
}

std::map<std::string, int> keywords;

void
initKeywordsMap()
{
	keywords["SELECT"] = SELECT;
	keywords["INSERT"] = INSERT;
	keywords["UPDATE"] = UPDATE;
	keywords["DELETE"] = DELETE;
	keywords["FROM"] = FROM;
	keywords["WHERE"] = WHERE;
	keywords["ORDER"] = ORDER;
	keywords["BY"] = BY;
	keywords["VALUES"] = VALUES;
	keywords["ASC"] = ASC;
	keywords["DESC"] = DESC;
	keywords["INTO"] = INTO;
	keywords["IN"] = IN;
	keywords["LIKE"] = LIKE;
	keywords["IS"] = IS;
	keywords["BETWEEN"] = BETWEEN;
	keywords["REGEXP"] = REGEXP;
	keywords["AND"] = AND;
	keywords["OR"] = OR;
	keywords["NOT"] = NOT;
}

bool eof = false;
std::string cmd;
std::string::iterator p;

void
initLexer(std::string s)
{
	cmd = s;
	p = cmd.begin();
}

std::string::value_type
getc()
{
	if (p == cmd.end()) {
		eof = true;
		return -1;
	}
	return *p++;
}

void
ungetc()
{
	if (!eof) {
		eof = false;
		p--;
	}
}

int
follow(int c, int ifyes, int ifno)
{
	if (getc()==c) {
		return ifyes;
	}
	ungetc();
	return ifno;
}

std::string::value_type
backslash(std::string::value_type c)
{
	if (c != '\\') {
		return c;
	}
	c = getc();
	if (c == 'b') {
		return '\b';
	}
	if (c == 'f') {
		return '\f';
	}
	if (c == 'n') {
		return '\n';
	}
	if (c == 'r') {
		return '\r';
	}
	if (c == 't') {
		return '\t';
	}
	if (c == '\n') {
		return backslash(getc());
	}
	return c;
}

int
yylex_work()
{
	std::string::value_type c;
loop:
	c = getc();
	if (c < 0) {
		return 0;
	}
	if (c == ' ' || c == '\t' || c == '\n') {
		goto loop;
	}
	if (c == '\'') {
		std::string str;
		while ((c = getc()) != '\'') {
			str.push_back(backslash(c));
		}
		Node *n = new Node(VALUE_NODE);
		n->val.s= str;
		yylval.node = n;
		return STRING;
	}
	if ('0' <= c && c <= '9') {
		long l = c - '0';
		while('0' <= (c=getc()) && c <= '9') {
			l =10 * l + c - '0';
		}
		ungetc();
		Node *n = new Node(VALUE_NODE);
		n->val.i = l;
		yylval.node = n;
		return INTEGER;
	}
	if (('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z') || c == '_') {
		std::string name;
		do {
			name.push_back(c);
			c = getc();
		} while(('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z') || ('0' <= c && c <= '9') || c == '_');
		if (c > 0) {
			ungetc();
		}

		std::map<std::string, int>::iterator it = keywords.find(name);
		if (it != keywords.end()) {
			return it->second;
		}
		Node *n = new Node(VALUE_NODE);
		n->val.id = name;
		yylval.node = n;
		return ID;
	}
	switch(c){
	case '!':
		return follow('=', NE, '!');
	case '=':
		return EQ;
	case '>':
		return follow('=', GE, '>'); 
	case '<':
		if (follow('=', 1, 0)) {
			return LE;
		}
		if (follow('>', 1, 0)) {
			return NE;
		}
		return '<';
	}
	return c;
}

int
yylex()
{
	int i = yylex_work();
//	std::cout << "lex: " << i << "\n";
	return i;
}

int
main(int argc, char *argv[])
{
	initKeywordsMap();
	initLexer(argv[1]);
	return yyparse();
}
