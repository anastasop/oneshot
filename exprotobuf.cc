
#include <iostream>
#include <string>
#include <cassert>
#include <google/protobuf/descriptor.h>
#include "items.pb.h"

int
main(int argc, char *argv[])
{
	GOOGLE_PROTOBUF_VERIFY_VERSION;

	Items::IntItem ii;
	ii.set_item(10);
	std::cout << ii.DebugString();

	const google::protobuf::Descriptor* descriptor = ii.GetDescriptor();
	const google::protobuf::FieldDescriptor* item_field = descriptor->FindFieldByName("item");
	const google::protobuf::Reflection* reflection = ii.GetReflection();
	reflection->SetInt32(&ii, item_field, 12);
	std::cout << ii.DebugString();

	google::protobuf::ShutdownProtobufLibrary();
	return 0;
}
