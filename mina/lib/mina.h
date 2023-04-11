#include <stdarg.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdlib.h>

#define FIELD_SIZE 32

bool poseidon(uint8_t network_id,
              const uint8_t *field_ptr,
              uintptr_t field_len,
              uint8_t *output_ptr);

bool verify(uint8_t network_id,
            const uint8_t *pubkey_x,
            const uint8_t *pubkey_y,
            const uint8_t *sig_rx,
            const uint8_t *sig_s,
            const uint8_t *field_ptr,
            uintptr_t field_len,
            bool *output_ptr);
