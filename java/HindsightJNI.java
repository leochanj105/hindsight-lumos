import java.nio.ByteBuffer;
import java.nio.ByteOrder;

public class HindsightJNI{
  @Override
  public String toString() {
    return "HindsightJNI []";
  }
  public static native void hindsightInit(String procName, String config);
  //public static native void hindsightTracepoint(byte[] payload, int size);
  public static native Trace hindsightBegin(long traceId);
  public static native void returnBufferNative();
  public static native ByteBuffer switchBufferNative();
  // public static native void test(String x);
  public static void main(String args[]){
    System.out.println("initting...");
    // System.load("/home/jingyuan/hindsight/client/lib/libtracer.so");
    System.load("/home/jingyuan/hindsight/java/libHS.so");
    hindsightInit("hs_jnitest","/etc/hindsight_conf/default.conf");
    Trace t = hindsightBegin(123);
    int N =1000;
    int K =1000;

    byte[] bs = "abcd\n".getBytes();
    for (int i = 0; i < N; i++){
      t.hs_tracecpoint(bs);
    }
    t.returnBuffer();
  }


  //class TraceState{
  //  public Trace header;
  //  public Buffer buffer;
  //  //public TraceState()
  //}


  class Trace{
    // public long trace_id; //0
    // public long acquired; //8
    // public int buffer_id; //16
    // public int prev_buffer_id; //20
    // public int size; //24
    // public short buffer_number; //28
    // public short null_buffer_count; //30

    public ByteBuffer buffer;
    public ByteBuffer content;
    // public int remaining; // Space remaining in the underlying buffer

    public Trace(
        ByteBuffer buffer) {
      this.buffer = buffer;
      prepareBuffer();
    }

    public void prepareBuffer() {
      this.buffer.position(32);
      this.buffer.order(ByteOrder.LITTLE_ENDIAN);
      this.content = this.buffer.slice();
    }

    public void hs_tracecpoint(byte[] payload) {
      if(try_write(payload)) return;
      write(payload);
    }

    public boolean try_write(byte[] payload) {
      if(payload.length > content.remaining()) return false;
      this.content.put(payload);
      return true;
    }

    public void write(byte[]payload){
      //this.content.put(payload);
      int curr = 0;
      while(curr < payload.length){
        int remaining = content.remaining();
        int to_write = payload.length - curr;
        if(remaining >= to_write){
          content.put(payload, curr, to_write);
          curr = payload.length;
          break;
        }
        content.put(payload, curr, remaining);
        curr += remaining;
        switchBuffer();
      }
    }

    //public int writePartial(byte[]payload, )

    public void switchBuffer(){
      writeSizeToHeader();
      this.buffer = switchBufferNative();
      prepareBuffer();
    }

    public void writeSizeToHeader() {
      int size = content.capacity() - content.remaining() + 32;
      buffer.putInt(24, size);
    }

    public void returnBuffer(){
      writeSizeToHeader();
      returnBufferNative();
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
