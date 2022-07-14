add_library(cifuzz_internal_replayer STATIC EXCLUDE_FROM_ALL
            "${CMAKE_CURRENT_LIST_DIR}/../src/replayer.c")
if(MSVC)
  # TODO(fmeum): Remove once ASan has been stabilized and we no longer have to set /MTd as a compile option globally.
  target_compile_options(cifuzz_internal_replayer PUBLIC /MTd)
endif()
# If a CXX-only project depends on the replayer, it won't be able to compile and link the replayer's C source file.
# Using enable_language(C) from a package is discouraged, but we can work around this by asking the C++ compiler to
# compile the replayer as a C target if C is not enabled.
# https://discourse.cmake.org/t/is-it-appropriate-to-use-enable-language-in-a-cmake-package-file/4335/2
if(NOT C IN_LIST ENABLED_LANGUAGES)
  set_property(SOURCE "${CMAKE_CURRENT_LIST_DIR}/../src/replayer.c"
               APPEND
               PROPERTY LANGUAGE CXX)
  # Make the C++ compiler compile replayer.c as a C file without a warning such as:
  # clang: warning: treating 'c' input as 'c++' when in C++ mode, this behavior is deprecated
  if(MSVC)
    target_compile_options(cifuzz_internal_replayer PRIVATE /TC)
  else()
    target_compile_options(cifuzz_internal_replayer PRIVATE -x c)
  endif()
  set_target_properties(cifuzz_internal_replayer PROPERTIES LINKER_LANGUAGE CXX)
endif()
if(CIFUZZ_SANITIZERS)
  target_compile_definitions(cifuzz_internal_replayer PRIVATE CIFUZZ_HAS_SANITIZER)
endif()
