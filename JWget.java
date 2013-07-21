
import java.io.InputStream;
import java.io.OutputStream;
import java.net.HttpURLConnection;
import java.net.URL;

public class JWget {
	public static void main(String[] args) throws Exception {
		URL fetchURL = null;
		if (args.length > 0) {
			fetchURL = new URL(args[0]);
		} else {
			System.err.println("usage: JWget <url>");
			System.exit(2);
		}
	
		HttpURLConnection hurlc = null;
//		List<Proxy> proxies = proxySelector.select(u.toURI());
//		Proxy proxy = proxies.get(0);
//		hurlc = (HttpURLConnection) u.openConnection(proxy);
		hurlc = (HttpURLConnection) fetchURL.openConnection();
		hurlc.setDoInput(true);
		hurlc.setDoOutput(false);
		hurlc.setUseCaches(false);
		hurlc.setInstanceFollowRedirects(true);
//		hurlc.setRequestProperty("Content-Type", "text/xml;charset=UTF-8");
		hurlc.setRequestMethod("GET");
		hurlc.connect();

		int rcode = hurlc.getResponseCode();
		if (rcode != 200) {
			throw new Exception("Response status is not OK(200)");
		}

		InputStream ist = hurlc.getInputStream();
		byte[] buf = new byte[8192];
		int nr = 0;
		while((nr = ist.read(buf)) > 0) {
			System.out.write(buf, 0, nr);
		}
		ist.close();
	}
}
