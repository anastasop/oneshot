package jhug.anastasop.table;

import java.io.File;

import com.sleepycat.je.Cursor;
import com.sleepycat.je.CursorConfig;
import com.sleepycat.je.Database;
import com.sleepycat.je.DatabaseConfig;
import com.sleepycat.je.DatabaseEntry;
import com.sleepycat.je.Environment;
import com.sleepycat.je.EnvironmentConfig;
import com.sleepycat.je.OperationStatus;

public class BDBExample {
	public static void main(String[] args) throws Exception {
		/*
		Database environments:
			- encapsulate one or more DB
			- a single in-memory cache for each of the DBs of the environment
			- group operations against multiple DBs in a single transaction
			- administrative and configuration activities related to log files and the in-memory cache
		*/
		EnvironmentConfig cfg = new EnvironmentConfig();
		cfg.setAllowCreate(true);
		cfg.setReadOnly(false);
		cfg.setTransactional(false);
		Environment env = new Environment(new File("/home/spyros/storage/berkeleyDB/sandbox"), cfg);
		
		DatabaseConfig dcfg = new DatabaseConfig();
		dcfg.setAllowCreate(true);
		dcfg.setDeferredWrite(true);
		dcfg.setTransactional(false);
		Database db = env.openDatabase(null, "spydb", dcfg);
		
		DatabaseEntry key1 = new DatabaseEntry("key1".getBytes("UTF-8"));
		DatabaseEntry val1 = new DatabaseEntry("val1".getBytes("UTF-8"));
		OperationStatus putStatus1 = db.put(null, key1, val1);
		assert putStatus1 == OperationStatus.SUCCESS;
		DatabaseEntry key2 = new DatabaseEntry("key2".getBytes("UTF-8"));
		DatabaseEntry val2 = new DatabaseEntry("val2".getBytes("UTF-8"));
		OperationStatus putStatus2 = db.put(null, key2, val2);
		assert putStatus2 == OperationStatus.SUCCESS;
		db.sync();
		
		DatabaseEntry dval1 = new DatabaseEntry();
		OperationStatus getStatus1 = db.get(null, key1, dval1, null);
		assert getStatus1 == OperationStatus.SUCCESS;
		System.out.printf("%s: %s%n", new String(key1.getData(), "UTF-8"), new String(dval1.getData(), "UTF-8"));
		
		// create a cursor to iterate some records
		// cursors can also be used to search for keys with partial match and put/delete records
		CursorConfig ccfg = new CursorConfig();
		ccfg.setReadCommitted(true);
		ccfg.setReadUncommitted(false);
		Cursor cur = db.openCursor(null, ccfg);
		DatabaseEntry ckey = new DatabaseEntry();
		DatabaseEntry cval = new DatabaseEntry();
		while (cur.getNext(ckey, cval, null) == OperationStatus.SUCCESS) {
			System.out.printf("%s: %s%n", new String(ckey.getData(), "UTF-8"), new String(cval.getData(), "UTF-8"));
		}
		cur.close();
		
		db.delete(null, key1);
		db.delete(null, key2);
		
		db.close();
		env.close();
	}
}
