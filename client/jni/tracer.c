#include <jni.h>
#include "tracer.h"


#ifndef _Included_simple
#define _Included_simple
#ifdef __cplusplus
#include <iostream>
extern "C" {
#endif


JNIEXPORT jint JNICALL JNI_OnLoad(JavaVM *vm, void *reserved)
{
	TracerInit(100000, 1);

	return JNI_VERSION_1_1;
}

JNIEXPORT void JNICALL JNI_OnUnload(JavaVM *vm, void *reserved)
{
	TracerStop();	
}

/*
 * Class:     simple
 * Method:    TracerStop
 * Signature: ()V
 */
JNIEXPORT void JNICALL Java_simple_TracerStop
  (JNIEnv *env, jobject thisObject)
{
	TracerStop();
}

/*
 * Class:     simple
 * Method:    TracerUpdate
 * Signature: (J)V
 */
JNIEXPORT void JNICALL Java_simple_TracerUpdate
  (JNIEnv *env, jobject thisObject, jlong num)
{
	TracerUpdate (num); /* Note: jlong is signed 64-bit */
}
/*
 * Class:     simple
 * Method:    TracerSet
 * Signature: (JLjava/lang/String;J)V
 */
JNIEXPORT void JNICALL Java_simple_TracerSet
  (JNIEnv *env, jobject thisObject, jlong time, jstring location, jlong tpt_idx)
{
	TracerSet (time, "fixme", tpt_idx);
	// XXX: We should swap out "location" for just the tpt_idx and avoid all string marshalling 
	//const char *loc = (*env)->GetStringUTFChars(env, location, NULL); /* XXX: Could this get freed? uh-oh */
	//TracerSet (time, loc, tpt_idx);
}
/*
 * Class:     simple
 * Method:    TracerGetCurrReq
 * Signature: ()J
 */
JNIEXPORT jlong JNICALL Java_simple_TracerGetCurrReq
  (JNIEnv *env, jobject thisObject)
{
	return TracerGetCurrReq();
}



#ifdef __cplusplus
}
#endif
#endif
