public class A{
  public static native void hindsightInit(String procName, String config);
  public static native void test(String x);
  public static void main(String args[]){
    System.out.println("initting...");
    // System.load("/home/jingyuan/hindsight/client/lib/libtracer.so");
    System.load("/home/jingyuan/hindsight/java/libHS.so");
    hindsightInit("hs_jnitest","/etc/hindsight_conf/default.conf");
    test("asddd");
  }
}
