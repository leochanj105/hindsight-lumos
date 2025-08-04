public class HindsightJNI{
  public static native void hindsightInit(String procName, String config);
  public static native void hindsightTracepoint(byte[] payload, int size);
  public static native void hindsightBegin(long traceId);
  public static native void hindsightEnd();
  // public static native void test(String x);
  public static void main(String args[]){
    System.out.println("initting...");
    // System.load("/home/jingyuan/hindsight/client/lib/libtracer.so");
    System.load("/home/jingyuan/hindsight/java/libHS.so");
    hindsightInit("hs_jnitest","/etc/hindsight_conf/default.conf");
    hindsightBegin(123);
    byte[] pld = "hahaha".getBytes();
    hindsightTracepoint(pld, pld.length);
    hindsightEnd();
  }
}
