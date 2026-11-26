/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

/*
 * Start command of the default clidey/whodb image.
 *
 * On x86-64 the bundled DuckDB library is built for x86-64-v3 and uses AVX2,
 * BMI2 and FMA instructions. On a CPU without them the process is killed by an
 * illegal instruction while that library loads, before WhoDB can log anything,
 * so the check has to run before /core is started.
 *
 * The check executes one instruction of each kind and catches SIGILL. Asking
 * what the CPU advertises (CPUID, /proc/cpuinfo) is not reliable: Rosetta runs
 * these instructions without advertising them.
 */

#include <stdio.h>
#include <unistd.h>

#if defined(__x86_64__)
#include <setjmp.h>
#include <signal.h>

static sigjmp_buf illegal_instruction;

static void on_sigill(int sig) {
    (void)sig;
    siglongjmp(illegal_instruction, 1);
}

static int cpu_is_supported(void) {
    struct sigaction action = {.sa_handler = on_sigill};
    sigaction(SIGILL, &action, NULL);
    if (sigsetjmp(illegal_instruction, 1)) {
        return 0;
    }
    __asm__ volatile(
        "vpaddd %%ymm0, %%ymm0, %%ymm0\n"      /* AVX2 */
        "mulx %%rax, %%rcx, %%rdx\n"           /* BMI2 */
        "vfmadd132pd %%ymm0, %%ymm0, %%ymm0\n" /* FMA */
        "vzeroupper\n"
        : : : "ymm0", "rcx", "rdx");
    signal(SIGILL, SIG_DFL);
    return 1;
}
#endif

int main(int argc, char **argv) {
    (void)argc;
#if defined(__x86_64__)
    if (!cpu_is_supported()) {
        fputs("WhoDB cannot start: this CPU does not support AVX2, BMI2 and FMA.\n"
              "This image needs Intel Haswell, AMD Excavator or newer.\n"
              "Use the compatibility image instead: clidey/whodb:latest-compat\n",
              stderr);
        return 1;
    }
#endif
    execv("/core", argv);
    perror("WhoDB cannot start: /core");
    return 1;
}
