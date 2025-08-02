#include "A.h"
#include "stdio.h"

JNIEXPORT void JNICALL Java_A_hindsightInit
  (JNIEnv * env, jclass cls, jstring proc, jstring config){
    printf("start initing...\n");
    const char *procString = (*env)->GetStringUTFChars(env, proc, 0);
    const char *configString = (*env)->GetStringUTFChars(env, config, 0);
    hindsight_init_with_config(procString, 
        hindsight_load_config_file(configString)
        );
    printf("finished initting\n");
  }


JNIEXPORT void JNICALL Java_A_test
  (JNIEnv *env, jclass cls, jstring s){
    const char *nativeString = (*env)->GetStringUTFChars(env, s, 0);
    printf("%s\n", nativeString);
  }
