#include "HindsightJNI.h"
#include "stdio.h"
#include "hindsight.h"
//#include "hindsight.h"
#include "tracestate.h"
#include "buffer.h"
#include "error.h"
#include "assert.h"
JNIEXPORT void JNICALL Java_HindsightJNI_hindsightInit
  (JNIEnv * env, jclass cls, jstring proc, jstring config){
    printf("start initing...\n");
    const char *procString = (*env)->GetStringUTFChars(env, proc, 0);
    const char *configString = (*env)->GetStringUTFChars(env, config, 0);
    HindsightConfig c = hindsight_load_config_file(configString);

    hindsight_init_with_config(procString,c);
    printf("finished initting\n");
  }

JNIEXPORT void JNICALL Java_HindsightJNI_hindsightTracepoint
  (JNIEnv * env, jclass cls, jbyteArray payload, jint size){
    jboolean isCopy;
    /* printf("writing...\n"); */
    char * pCData = (char*)(*env)->GetByteArrayElements(env, payload, &isCopy);

    hindsight_tracepoint(pCData, size);
    if(isCopy)
    {
      (*env)->ReleaseByteArrayElements(env, payload, pCData, JNI_ABORT);
    }
  }

JNIEXPORT jobject JNICALL Java_HindsightJNI_hindsightBegin
  (JNIEnv * env, jclass cls, jlong traceId){
    hindsight_begin(traceId);
    jclass class = (*env)->FindClass(env,"HindsightJNI$Trace");
    
    if (NULL == class){
        printf("class");
        assert(0);
    }
    jmethodID mid = (*env)->GetMethodID(env,class, "<init>", "(LHindsightJNI;Ljava/nio/ByteBuffer;)V");

    if (NULL == mid){
       printf("method\n");
       assert(0);
    }

    printf("%p\n",hindsight_tls.buffer.base);
    return (*env)->NewObject(env, class, mid, 
        class,
        /* hindsight_tls.header.trace_id, */
        /* hindsight_tls.header.acquired, */
        /* hindsight_tls.header.buffer_id, */
        /* hindsight_tls.header.prev_buffer_id, */
        /* hindsight_tls.header.size, */
        /* hindsight_tls.header.buffer_number, */
        /* hindsight_tls.header.null_buffer_count, */
        (*env)->NewDirectByteBuffer(env,hindsight_tls.buffer.base, 
          hindsight_tls.buffer.remaining + hindsight_tls.buffer.ptr - hindsight_tls.buffer.base));
        /* hindsight_tls.buffer.remaining */
  }

JNIEXPORT jobject JNICALL Java_HindsightJNI_switchBufferNative
  (JNIEnv *env, jclass cls){
    switchBuffer(&hindsight_tls, mgr);
    printf("new buffer: %p\n",hindsight_tls.buffer.base);
    return (*env)->NewDirectByteBuffer(env,hindsight_tls.buffer.base,
      hindsight_tls.buffer.remaining + hindsight_tls.buffer.ptr - hindsight_tls.buffer.base);
  }

JNIEXPORT void JNICALL Java_HindsightJNI_returnBufferNative
  (JNIEnv *env, jclass cls){
    bufmanager_return(mgr, hindsight_tls.header.trace_id, &hindsight_tls.buffer);
  }


JNIEXPORT void JNICALL Java_HindsightJNI_hindsightEnd
  (JNIEnv *env, jclass cls){
    hindsight_end();
  }

