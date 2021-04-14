import java.lang.*;

public class simple {
 
    static {
    	/* Calls JNI_OnLoad() */
        System.loadLibrary("jnitracer");
    }
    
    public static void main(String[] args) {
    	final int iterations = 100000000;

        System.out.println ("From Java app");
	simple tracer = new simple();
	tracer.TracerUpdate(500);
	long start = System.nanoTime();
	for (int i=0; i < iterations; i++)
	{
		//System.out.println ("About to call another tracepoint");
		tracer.TracerSet(100+i, "Testing string", 300+i);	
	}
	long end = System.nanoTime();
	System.out.println ("Average trace call: " + ((end-start)/iterations) + " ns/it");
    }
 
    public native void TracerStop();
    public native void TracerUpdate(long num);
    public native void TracerSet(long time, String location, long tpt_idx);
    public native long TracerGetCurrReq();

}
