public class simple {
 
    static {
        System.loadLibrary("tracer");
    }
    
    public static void main(String[] args) {
        System.out.println ("From Java app");
        new simple().trace();
    }
 
    private native void trace();
}
