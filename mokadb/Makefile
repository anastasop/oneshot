
all: mokadb test

mokadb: mokasql.tab.c
	g++ -Wall -o mokadb mokasql.tab.c

mokasql.tab.c: mokasql.y
	bison mokasql.y

clean:
	rm -f mokasql.tab.* mokadb

test: mokadb
	./mokadb "SELECT * FROM SPY"
	./mokadb "SELECT * FROM SPY WHERE pubs > 10"
	./mokadb "SELECT * FROM SPY WHERE 2 < 10"
	./mokadb "SELECT * FROM SPY WHERE name IN ('A', 'B')"
	./mokadb "INSERT INTO SPY VALUES (1, 2, 3)"

