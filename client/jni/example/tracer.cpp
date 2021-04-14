#include <jni.h>
#include <iostream>

#ifndef _Included_simple
#define _Included_simple
#ifdef __cplusplus
extern "C" {
#endif
/*
 * Class:     simple
 * Method:    trace
 * Signature: ()V
 */
JNIEXPORT void JNICALL Java_simple_trace(JNIEnv *env, jobject thisObject)
{
	std::cout << "Hello from C++ world " << std::endl;
}

#ifdef __cplusplus
}
#endif
#endif
