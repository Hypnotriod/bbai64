package titfldelegate

/*
#ifndef TI_TFL_EXTERNAL_DELEGATE_H_
#define TI_TFL_EXTERNAL_DELEGATE_H_

#define _GNU_SOURCE

#include <stdio.h>
#include <stdarg.h>
#include <stdlib.h>
#include <tensorflow/lite/c/c_api.h>
#include "tensorflow/lite/c/common.h"

#cgo CFLAGS: -std=c99
#cgo CXXFLAGS: -std=c99

#define kExternalDelegateMaxOptions 256
typedef struct TfLiteExternalDelegateOptions {
  const char* lib_path;
  int count;
  const char* keys[kExternalDelegateMaxOptions];
  const char* values[kExternalDelegateMaxOptions];
  TfLiteStatus (*insert)(struct TfLiteExternalDelegateOptions* options,
                         const char* key, const char* value);
} TfLiteExternalDelegateOptions;

TfLiteStatus TfLiteExternalDelegateOptionsInsert(
    TfLiteExternalDelegateOptions* options, const char* key, const char* value);

TfLiteExternalDelegateOptions TfLiteExternalDelegateOptionsDefault(
    const char* lib_path);

TfLiteDelegate* TfLiteExternalDelegateCreate(
    const TfLiteExternalDelegateOptions* options);

void TfLiteExternalDelegateDelete(TfLiteDelegate* delegate);

TfLiteDelegate* TiTflDelegateCreate(const char* delegate_so, const char* artifacts_folder)
{
	TfLiteExternalDelegateOptions options = TfLiteExternalDelegateOptionsDefault(delegate_so);
	TfLiteExternalDelegateOptionsInsert(&options, "tidl_tools_path", "null");
	TfLiteExternalDelegateOptionsInsert(&options, "import", "no");
	TfLiteExternalDelegateOptionsInsert(&options, "artifacts_folder", artifacts_folder);
	return TfLiteExternalDelegateCreate(&options);
}

#endif  // TI_TFL_EXTERNAL_DELEGATE_H_
*/
import "C"
import (
	"unsafe"

	"github.com/mattn/go-tflite/delegates"
)

type TiTflDeleg struct {
	d *C.TfLiteDelegate
}

// Delete the delegate
func (d *TiTflDeleg) Delete() {
	C.TfLiteExternalDelegateDelete(d.d)
}

// Return a pointer
func (d *TiTflDeleg) Ptr() unsafe.Pointer {
	return unsafe.Pointer(d.d)
}

func TiTflDelegateCreate(libPath string, artifactsPath string) delegates.Delegater {
	d := C.TiTflDelegateCreate(C.CString(libPath), C.CString(artifactsPath))
	if d == nil {
		return nil
	}
	return &TiTflDeleg{
		d: d,
	}
}
