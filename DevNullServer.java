
import java.io.FileOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.net.InetSocketAddress;
import java.util.Date;

import com.sun.net.httpserver.HttpExchange;
import com.sun.net.httpserver.HttpHandler;
import com.sun.net.httpserver.HttpServer;

class DevNullHandler implements HttpHandler {
	@Override
	public void handle(HttpExchange exch) throws IOException {
		System.out.println(new Date());
		InputStream ist = exch.getRequestBody();
		byte[] buf = new byte[8192];
		int nr = 0;
		try {
			while ((nr = ist.read(buf, 0, buf.length)) > 0) {
				System.out.write(buf, 0, nr);
			}
		} finally {
			ist.close();
		}
		System.out.print("\n");
		exch.sendResponseHeaders(203, 0);
		exch.close();
	}
}

public class DevNullServer {
	public static void main(String[] args) throws Exception {
		String ctxtPath = args.length == 0? "/": args[0];
		HttpServer server = HttpServer.create(new InetSocketAddress(65101), 5);
		server.createContext(ctxtPath, new DevNullHandler());
		server.setExecutor(null); // creates a default executor
		server.start();
	}
}
