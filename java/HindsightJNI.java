import java.nio.ByteBuffer;
import java.nio.ByteOrder;

public class HindsightJNI{
  @Override
  public String toString() {
    return "HindsightJNI []";
  }
  public static native void hindsightInit(String procName, String config);
  public static native void hindsightTracepoint(byte[] payload, int size);
  public static native Trace hindsightBegin(long traceId);
  public static native void hindsightEnd();
  public static native void returnBufferNative();
  public static native ByteBuffer switchBufferNative();
  // public static native void test(String x);
  public static void main(String args[]){
    System.out.println("initting...");
    // System.load("/home/jingyuan/hindsight/client/lib/libtracer.so");
    System.load("/home/jingyuan/hindsight/java/libHS.so");
    hindsightInit("hs_jnitest","/etc/hindsight_conf/default.conf");
    Trace t= hindsightBegin(11);
    byte[] bs = new byte[]{0,1,2,3};
    int x = 1231;


    for (int i = 0; i < 100000; i++){
      // t.hs_tracepoint(12,10L);
      t.hs_tracecpoint(bs);
    }

    // t.returnBuffer();
    hindsightEnd();


    t = hindsightBegin(123);

    int N = 1000;
    String ss = System.getProperty("N");
    if (ss != null) {
      N = Integer.valueOf(ss);
    }
    int K =1000;
    System.out.println("N:"+N);
    //byte[] bs = "abcde\n".getBytes();

    long start = System.nanoTime();
    for (int i = 0; i < N; i++){
      // hindsightTracepoint(bs, 4);
      t.hs_tracecpoint(bs);
      // t.hs_tracepoint(10,100L);
      // t.hs_tracepoint(10,100L);
      // t.hs_tracepoint(10,100L);
      // t.hs_tracepoint(10,100L);
      // t.hs_tracepoint(10,100L);
      // t.hs_tracepoint(10,100L);
      // t.hs_tracepoint(10,100L);
      // t.hs_tracepoint(10,100L);
      // t.hs_tracepoint(10,100L);
      // t.hs_tracepoint(10,100L);
      // System.arraycopy(bs, 0, t.cache, 0, 4);
      // t.content.putInt(i);
    }
    long end = System.nanoTime();
    System.out.println((end-start)/N/1);
    //t.returnBuffer();
    // t.returnBuffer();
    hindsightEnd();
  }


  //class TraceState{
  //  public Trace header;
  //  public Buffer buffer;
  //  //public TraceState()
  //}
  //
  public class Record{
    public int id;
    public long addr;
    public Record(int id,long addr){
      this.id = id;
      this.addr = addr;
    }
  }

  class Trace{
    // public long trace_id; //0
    // public long acquired; //8
    // public int buffer_id; //16
    // public int prev_buffer_id; //20
    // public int size; //24
    // public short buffer_number; //28
    // public short null_buffer_count; //30
    public static final int BSIZE = 4096 - 32;
    public int position = 0;
    public byte[] cache = new byte[BSIZE];
    public Record[] rec = new Record[1024];
    public ByteBuffer buffer;
    public ByteBuffer content;
    // public int remaining; // Space remaining in the underlying buffer

    public int remaining(){
      return cache.length - position;
    }

    public Trace(ByteBuffer buffer) {
      this.buffer = buffer;
      prepareBuffer();
    }

    public void prepareBuffer() {
      position = 0;
      //this.buffer.position(32);
      //this.buffer.order(ByteOrder.LITTLE_ENDIAN);
      //this.content = this.buffer.slice();
    }

    public void hs_tracecpoint(byte[] payload) {
      if(try_write(payload)) return;
      write(payload);
    }

    public final void hs_tracepoint(int id, long payload){

      // rec[position] = new Record(id, payload);
      // position = (position+1)%rec.length;
      // if(position >= rec.length){}
      
      int size = 12;
      if (size > (cache.length-position)) {
        switchBuffer();
        position = 0;
      }
      cache[position] = (byte) id;
      cache[position+1] = (byte) id;
      cache[position+2] = (byte) id;
      cache[position+3] = (byte) id;
      cache[position+4] = (byte) id;
      cache[position+5] = (byte) id;
      cache[position+6] = (byte) id;
      cache[position+7] = (byte) id;
      cache[position+8] = (byte) id;
      cache[position+9] = (byte) id;
      cache[position+10] = (byte) id;
      cache[position+11] = (byte) id;
      position += size;
    }

    public boolean try_write(byte[] payload) {
      if(payload.length > remaining()) return false;
      //this.content.put(payload);
      System.arraycopy(payload, 0, cache, position, payload.length);
      position += payload.length;
      return true;
    }

    public void write(byte[]payload){
      int curr = 0;
      while(curr < payload.length){
        int remaining = remaining();//content.remaining();
        int to_write = payload.length - curr;
        if(remaining >= to_write){
          //content.put(payload, curr, to_write);
          System.arraycopy(payload, curr, cache, position, to_write);
          position += to_write;
          curr = payload.length;
          break;
        }
        //content.put(payload, curr, remaining);
        System.arraycopy(payload, curr, cache, position, remaining);
        position += remaining;
        curr += remaining;
        switchBuffer();
      }
    }


    public void switchBuffer(){
      // System.out.println("switching...");
      endBuffer();
      //this.buffer = switchBufferNative();
      prepareBuffer();
    }

    public void endBuffer(){
      hindsightTracepoint(cache, position);
      //writeSizeToHeader();
      //content.put(cache, 0, position);
    }
    public void writeSizeToHeader() {
      //int size = content.capacity() - content.remaining() + 32;
      int size = 32 + position;
      buffer.putInt(24, size);
    }

    public void returnBuffer(){
      endBuffer();
      // returnBufferNative();
    }

    @Override
    public String toString() {
      return "Trace [buffer=" + buffer + "]";
    }


  }

  //class Buffer{
  //  public int id; // Equivalent to the index of this buffer in the buffer pool
  //  public int remaining; // Space remaining in the underlying buffer
  //  //public byte[] ptr; // Pointer to next available byte in buffer
  //  //public int current;
  //  ByteBuffer buf;
  //}
}
