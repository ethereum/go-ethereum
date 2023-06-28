#include <stdarg.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdlib.h>

#define FIELD_SIZE 32

/**
 *  * # Safety  * this functions accepts raw pointer from golang
 */
bool poseidon(uint8_t network_id,
              const uint8_t *field_ptr,
              uintptr_t field_len,
              uint8_t *output_ptr);

/**
 *  * # Safety  * this functions accepts raw pointer from golang
 */
bool verify(uint8_t network_id,
            const uint8_t *pubkey_x,
            const uint8_t *pubkey_y,
            const uint8_t *sig_rx,
            const uint8_t *sig_s,
            const uint8_t *field_ptr,
            uintptr_t field_len,
            bool *output_ptr);

/**
 *  * # Safety  * this functions accepts raw pointer from golang
 */
bool transaction_commitment(const uint8_t *zkapp_command_ptr,
                            uintptr_t zkapp_command_len,
                            uint8_t *output_ptr);
