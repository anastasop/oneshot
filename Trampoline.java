
import java.io.*;
import java.net.*;
import java.awt.*;
import java.awt.event.*;
import javax.swing.*;

class Tramwin extends JFrame {
	private JPanel ptop;
	private JTextArea tfr;
	private JScrollPane tsp;

	Tramwin(){
		setTitle("SOAP viewer");
		setResizable(false);

		tfr = new JTextArea(40, 80);
		tfr.setLineWrap(true);
		tsp = new JScrollPane(tfr,
			JScrollPane.VERTICAL_SCROLLBAR_ALWAYS,
			JScrollPane.HORIZONTAL_SCROLLBAR_ALWAYS
		);
		ptop = new JPanel(new BorderLayout());
		ptop.add(tsp);

		Container cp = getContentPane();
		cp.add(ptop, BorderLayout.CENTER);
		pack();
		setVisible(true);
	}

	public void addText(String s){
		tfr.append(s);
	}

	public static void main(String[] args){
		Tramwin tw = new Tramwin();
	}
}

class Xfer implements Runnable {
	private InputStream from;
	private OutputStream to;
	private Tramwin log;

	public Xfer(InputStream from, OutputStream to, Tramwin log){
		this.from = from;
		this.to = to;
		this.log = log;
	}

	public void run(){
		try{
			int n = 0;
			byte[] buf = new byte[8192];
			while((n = from.read(buf)) > 0){
				log.addText(new String(buf, 0, n));
				if(to != null) {
					to.write(buf, 0, n);
					to.flush();
				}
			}
			if (to != null && n == 0)
				to.close();
		}catch(IOException e){
			System.err.println("IOException: " + e.getMessage());
		}
		finally{
			//System.exit(2);
		}
	}
}

public class Trampoline {
	public static void main(String[] args) throws Exception {
		if(args.length != 3){
			System.err.println("trampoline: listenport ip port");
			System.exit(2);
		}
		String lport = args[0];
		String ipaddr = args[1];
		String port = args[2];

		ServerSocket ear = new ServerSocket(Integer.parseInt(lport));
		for(;;){
			Socket from = ear.accept();
			Socket to = new Socket(ipaddr, Integer.parseInt(port));
			Tramwin tw = new Tramwin();
			Thread t1 = new Thread(new Xfer(from.getInputStream(), to.getOutputStream(), tw));
			t1.start();
			Thread t2 = new Thread(new Xfer(to.getInputStream(), from.getOutputStream(), tw));
			t2.start();
		}
	}
}
