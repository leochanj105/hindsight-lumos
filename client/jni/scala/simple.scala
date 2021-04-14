//import com.sun.jna.{Library, Native, Platform}
//import scala.scalanative.native._


/*
trait Tracer extends Library {
    def TracerInit(size : Int, clear : Int) : Unit;
    def TracerStop() : Unit;
    def TracerUpdate(num : Int) : Unit;
    def TracerSet(time : Int, location : String, tpt_idx : Int) : Unit;
    def TracerGetCurrReq() : Int;
}*/

class simple {
   //@native def TracerInit(size : Long, clear : Long) : Unit;
   @native def TracerStop() : Unit;
   @native def TracerUpdate(num : Long) : Unit;
   @native def TracerSet(time : Long, location : String, tpt_idx : Long) : Unit;
   @native def TracerGetCurrReq() : Long;
}


object simple {
  // XXX upgrade to NativeLoader.load at some point
  //def Instance = Native.loadLibrary("simple", classOf[Tracer]).asInstanceOf[Tracer];
  //System.loadLibrary("jnitracer")
}


object TracerTest {
  def main(args: Array[String]) : Unit = {
    val iterations = 10000000;
    System.loadLibrary("jnitracer");
    //System.loadLibrary("simple");
    //val iterations = 10;
    var tracer = new simple; //.Instance;
    //tracer.TracerInit(10000, 1);
    println ("Tracer.Instance");
    tracer.TracerUpdate(500);
    println ("nanotime");
    var start = System.nanoTime();
    //val where : CString = c"Testing string";
    var tst = 1;
    for (it <- 1 to iterations) {
      tracer.TracerSet(100+it, "Testing string", 300+it);
      //tracer.TracerGetCurrReq();
      //tst = it+5;
      //"Argument %d: %s".format(i.asInstanceOf[AnyRef])

    }
    var end = System.nanoTime();
    println("Average trace call " + ((end-start)/iterations) + " ns/req");
  }
}

