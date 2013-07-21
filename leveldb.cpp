
#include <iostream>
#include "leveldb/db.h"

int
main()
{
	leveldb::Options options;
	options.create_if_missing = true;
	leveldb::DB* db;
	leveldb::Status status = leveldb::DB::Open(options, "/tmp/testdb", &db);
	assert(status.ok());

	leveldb::Status s;
	s = db->Put(leveldb::WriteOptions(), "Hello", "World");
	assert(s.ok());
	std::string val;
	s = db->Get(leveldb::ReadOptions(), "Hello", &val);
	assert(s.ok());

	std::cout << val << std::endl;

	delete db;
}
