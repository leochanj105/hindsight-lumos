#include "JNI.h"
#include "stdio.h"
JNIEXPORT void JNICALL Java_A_hindsightInit
  (JNIEnv * env, jclass cls, jstring s1, jstring s2){

  }


JNIEXPORT void JNICALL Java_A_test
  (JNIEnv *env, jclass cls, jstring s){
      printf("%s\n", (*env)->GetStringUTFChars(s ,0));
}
