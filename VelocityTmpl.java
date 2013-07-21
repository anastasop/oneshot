
import java.io.*;
import org.apache.velocity.VelocityContext;
import org.apache.velocity.Template;
import org.apache.velocity.app.Velocity;

public class Generator {
	public static void main(String[] args) throws Exception {
		Velocity.init();
		VelocityContext context = new VelocityContext();
		Template jibxTemplate = Velocity.getTemplate("./Templates/JiBXTemplate.vm");
		Template hibernateTemplate = Velocity.getTemplate("./Templates/HibernateTemplate.vm");
		Template hibernateDescriptionTemplate = Velocity.getTemplate("./Templates/hibernateDescriptionTemplate.vm");
		Template sqlCreateTemplate = Velocity.getTemplate("./Templates/sqlCreateTemplate.vm");
		Template sqlDropTemplate = Velocity.getTemplate("./Templates/sqlDropTemplate.vm");
//		Template sqlEmptyTemplate = Velocity.getTemplate("./Templates/sqlEmptyTemplate.vm");
		
		BufferedReader r = new BufferedReader(new FileReader("./metadata.txt"));
		String line = null;
		while ((line = r.readLine()) != null) {
			if (line.length() == 0 || line.charAt(0) == '#') {
				continue;
			}
			
			String[] parts = line.trim().split("[ \t]+");
			if (parts.length != 6) {
				throw new RuntimeException("Badly formatted line: " + line);
			}
			String description = parts[0];
			String tableName = parts[1];
			String primaryTrigger = parts[2];
			String secondaryTrigger = parts[3];
			String sqlType = parts[4];
			String elementName = parts[5];
			
			context.put("description", description);
			context.put("tableName", tableName);
			context.put("primaryTrigger", primaryTrigger);
			context.put("secondaryTrigger", secondaryTrigger);
			context.put("sqlType", sqlType);
			context.put("elementName", elementName);
			
			Writer w = null;
			w = new FileWriter("./JiBX/" + description + ".jibx.xml");
			jibxTemplate.merge(context, w);
			w.close();
			w = new FileWriter("./Hibernate/" + description + ".hbm.xml");
			hibernateTemplate.merge(context, w);
			w.close();
			w = new FileWriter("./Hibernate/" + description + "Description.hbm.xml");
			hibernateDescriptionTemplate.merge(context, w);
			w.close();
			
			w = new FileWriter("./SQLCreate/" + "create" + description + ".sql");
			sqlCreateTemplate.merge(context, w);
			w.close();
			w = new FileWriter("./SQLDrop/" + "drop" + description + ".sql");
			sqlDropTemplate.merge(context, w);
			w.close();
//			w = new FileWriter("./SQLEmpty/" + "empty" + description + ".sql");
//			sqlEmptyTemplate.merge(context, w);
//			w.close();
		}
		r.close();		
	}
}
