
d1 = new Packages.java.util.Date();
d2 = new java.util.Date();

print(d1);
print(d2);

// javascript also has a Date as Boolean, String etc
d3 = new Date();
print(d3); 

importPackage(java.io);
f = new File("/home/spyros");
print(f.getName() + " " + f.exists());

// access field, methods, properties as usual
// obj.name as a shortcut for get/setName()
// for overloaded functions resolution at runtime

// implementing java interfaces
obj = { run: function() { print("running\n"); }};
r = new java.lang.Runnable(obj);
t = new java.lang.Thread(r);
t.start();

// alternative method
// this way multiple interfaces can be implemented
k = new JavaAdapter(java.lang.Runnable, obj);
//k = new JavaAdapter(java.lang.Runnable, ..., obj);
t = new java.lang.Thread(r);
t.start();

// java arrays need special handling (reflection)
a = java.lang.reflect.Array.newInstance(java.lang.String, 5);
a[2] = 'monkey';
print(a[2]);
// arrays of primitives are more special
a = java.lang.reflect.Array.newInstance(java.lang.Integer.TYPE, 5);
a[1] = 3;
print(a[1]);


// strings can be user interchangeable
s1 = new java.lang.String("java");
s2 = "javascript";
print(s1.length()); print(s2.length);
// javascript method are available to java script if undefined
print(s1.match(/a.*/));

// exceptions are wrapped into object with the properties
// javaException rhinoException
try { 
  java.lang.Class.forName("NonExistingClass"); 
} catch (e) {
  if (e.javaException instanceof java.lang.ClassNotFoundException) {
    print("Class not found");
  }
}

