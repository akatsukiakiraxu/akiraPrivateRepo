#include <stdio.h>
#include <stdlib.h>
#include <memory.h>
#include <stdint.h>
#include <sys/types.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <unistd.h>

static const size_t MAX_PAYLOAD_SIZE = 32768;
static const uint32_t SIGNATURE = 0x12345678;
typedef struct {
    uint32_t signature;
    uint32_t type;
    uint32_t flags;
    uint32_t length;
} PacketHeader;

int main(int argc, char* argv[])
{
    if( argc < 4 ) {
        fprintf(stderr, "Usage: %s <input> <ipaddr> <port>\n", argv[0]);
        return 1;
    } 

    const char* input_path = argv[1];
    const char* address    = argv[2];
    const char* port       = argv[3];

    // Construct sockaddr_in
    struct sockaddr_in addr;
    memset(&addr, 0, sizeof(addr));
    addr.sin_family = AF_INET;
    addr.sin_port = htons(atoi(port));
    inet_aton(address, &addr.sin_addr);

    // Open file
    FILE* file = fopen(input_path, "rb");
    if( file == NULL ) {
        fprintf(stderr, "Error: failed to open input file - %s.", input_path);
        return 1;
    }
    
    if( fseek(file, 0, SEEK_END) != 0 ) {
        fclose(file);
        fprintf(stderr, "Error: failed to determine input size.");
        return 2;
    }
    size_t input_size = ftell(file);
    fseek(file, 0, SEEK_SET);

    // Allocate packet buffer.
    void* packet_buffer = malloc(MAX_PAYLOAD_SIZE + sizeof(PacketHeader) + 4);
    if( packet_buffer == NULL ) {
        fclose(file);
        fprintf(stderr, "Error: Failed to allocate buffer.");
        return 2;
    }

    // Create socket.
    int conn = socket(AF_INET, SOCK_DGRAM, 0);
    if( conn < 0 ) {
        fclose(file);
        fprintf(stderr, "Error: failed to create socket.");
        return 2;
    }

    // Initialize packet header.
    PacketHeader* header_ptr = (PacketHeader*)packet_buffer;
    uint32_t* sequence_ptr = (uint32_t*)((uintptr_t)packet_buffer + sizeof(PacketHeader));
    void* payload_head = (void*)((uintptr_t)packet_buffer + sizeof(PacketHeader) + 4 );
    header_ptr->signature = SIGNATURE;
    header_ptr->type = 0;

    // Send whole data.
    for(size_t bytes_transferred = 0; bytes_transferred < input_size; bytes_transferred += MAX_PAYLOAD_SIZE ) {
        size_t bytes_to_transfer = input_size - bytes_transferred;
        if( bytes_to_transfer > MAX_PAYLOAD_SIZE ) {
            bytes_to_transfer = MAX_PAYLOAD_SIZE;
        }
        // Fill header fields.
        header_ptr->flags = 0;
        header_ptr->flags |= bytes_transferred == 0 ? 1 : 0;
        header_ptr->flags |= bytes_transferred + bytes_to_transfer == input_size ? 2 : 0;
        header_ptr->length = bytes_to_transfer;
        // Read waveform data to the packet buffer.
        fread(payload_head, bytes_to_transfer, 1, file);

        // Send packet
        sendto(conn, packet_buffer, bytes_to_transfer + sizeof(PacketHeader) + 4, 0, (struct sockaddr*)&addr, sizeof(addr));

        // Increment sequence number
        (*sequence_ptr)++;
    }
    close(conn);

    return 0;
}