#include <glog/logging.h>

int main(int argc, char* argv[]) {
	google::InitGoogleLogging(argv[0]);
	google::InstallFailureSignalHandler();

	LOG(WARNING) << "gflags in not installed";
	LOG(INFO) << "glog is working";
	LOG_IF(INFO, 1 < 5) << "1 is smaller than 5";
	for (int i = 0; i < 4; i++) {
		LOG_EVERY_N(INFO, 2) << "this appears only 2 times although is in a #4 for";
	}
	// LOG_IF_EVERY_N(INFO, (size > 1024), 10)
	for (int i = 0; i < 4; i++) {
		LOG_FIRST_N(INFO, 2) << "log only first occurences " << google::COUNTER ;
	}

	DLOG(INFO) << "shown only in debugging, no -NDEBUG";


	// checks. unlike assert() there are not controlled by NDEBUG
	CHECK(1 < 5) << "comparison failed";
	CHECK_EQ(1, 1) << "comparison failed";

	// Google Style perror()
	// PLOG, PLOG_IF, PCHECK append a description of errno to output lines


	// Verbose Logging
	// define your own numeric logging levels for very verbose logging
	// useful during debugging. The log at INFO level
	// can be set differently for each module(file) with --vmodule
	// The argument has to contain a comma-separated list of <module name>=<log level>
	// <module name> is a glob pattern (e.g., gfs* for all modules whose name starts with "gfs"),
	// matched against the filename base (that is, name ignoring .cc/.h./-inl.h) 
	VLOG(1) << "I'm printed when you run the program with --v=1 or higher";
	VLOG(2) << "I'm printed when you run the program with --v=2 or higher";
	// also provided VLOG_IS_ON(), VLOG_IF(), VLOG_EVERY(), VLOG_IF_EVERY()
}

// Severity Level INFO, WARNING, ERROR, FATAL
// Setting Flags
// Conditional / Occasional Logging LOG_IF, LOG_EVERY
// Debug mode support DLOG
// CHECK Macros CHECK, CHECK_EQ, CHECK_STREQ
// Verbose Logging
// Failure Signal Handler

