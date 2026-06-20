#include "png_writer.hpp"

#include "io.hpp"

#include <algorithm>
#include <array>
#include <cstdint>
#include <fstream>
#include <stdexcept>
#include <vector>

namespace lfdraw {

static void write_u32_be(std::ostream& output, std::uint32_t value) {
    output.put(static_cast<char>((value >> 24) & 0xff));
    output.put(static_cast<char>((value >> 16) & 0xff));
    output.put(static_cast<char>((value >> 8) & 0xff));
    output.put(static_cast<char>(value & 0xff));
}

static void append_u32_be(std::vector<std::uint8_t>& out, std::uint32_t value) {
    out.push_back(static_cast<std::uint8_t>((value >> 24) & 0xff));
    out.push_back(static_cast<std::uint8_t>((value >> 16) & 0xff));
    out.push_back(static_cast<std::uint8_t>((value >> 8) & 0xff));
    out.push_back(static_cast<std::uint8_t>(value & 0xff));
}

static std::uint32_t update_crc(std::uint32_t crc, std::uint8_t byte) {
    crc ^= byte;
    for (int i = 0; i < 8; ++i) {
        crc = (crc & 1U) != 0 ? 0xedb88320U ^ (crc >> 1U) : crc >> 1U;
    }
    return crc;
}

static std::uint32_t crc32(const char type[4], const std::vector<std::uint8_t>& data) {
    std::uint32_t crc = 0xffffffffU;
    for (int i = 0; i < 4; ++i) {
        crc = update_crc(crc, static_cast<std::uint8_t>(type[i]));
    }
    for (const auto byte : data) {
        crc = update_crc(crc, byte);
    }
    return crc ^ 0xffffffffU;
}

static std::uint32_t adler32(const std::vector<std::uint8_t>& data) {
    constexpr std::uint32_t mod = 65521;
    std::uint32_t a = 1;
    std::uint32_t b = 0;
    for (const auto byte : data) {
        a = (a + byte) % mod;
        b = (b + a) % mod;
    }
    return (b << 16U) | a;
}

static std::vector<std::uint8_t> png_scanlines(const Image& image) {
    std::vector<std::uint8_t> raw;
    raw.reserve(static_cast<std::size_t>(image.height) * (static_cast<std::size_t>(image.width) * 3 + 1));
    for (int y = 0; y < image.height; ++y) {
        raw.push_back(0); // PNG filter type 0: none.
        for (int x = 0; x < image.width; ++x) {
            const auto& pixel = image.pixels[static_cast<std::size_t>(y * image.width + x)];
            raw.push_back(pixel.r);
            raw.push_back(pixel.g);
            raw.push_back(pixel.b);
        }
    }
    return raw;
}

static std::vector<std::uint8_t> zlib_store(const std::vector<std::uint8_t>& raw) {
    std::vector<std::uint8_t> out;
    out.reserve(raw.size() + raw.size() / 65535 * 5 + 16);
    out.push_back(0x78); // zlib header: deflate, 32K window.
    out.push_back(0x01); // fastest/no compression check bits.
    std::size_t offset = 0;
    while (offset < raw.size()) {
        const std::size_t remaining = raw.size() - offset;
        const std::uint16_t block_size = static_cast<std::uint16_t>(std::min<std::size_t>(remaining, 65535));
        const bool final = offset + block_size == raw.size();
        out.push_back(final ? 0x01 : 0x00);
        out.push_back(static_cast<std::uint8_t>(block_size & 0xff));
        out.push_back(static_cast<std::uint8_t>((block_size >> 8) & 0xff));
        const std::uint16_t nlen = static_cast<std::uint16_t>(~block_size);
        out.push_back(static_cast<std::uint8_t>(nlen & 0xff));
        out.push_back(static_cast<std::uint8_t>((nlen >> 8) & 0xff));
        out.insert(out.end(), raw.begin() + static_cast<std::ptrdiff_t>(offset), raw.begin() + static_cast<std::ptrdiff_t>(offset + block_size));
        offset += block_size;
    }
    append_u32_be(out, adler32(raw));
    return out;
}

static void write_chunk(std::ostream& output, const char type[4], const std::vector<std::uint8_t>& data) {
    write_u32_be(output, static_cast<std::uint32_t>(data.size()));
    output.write(type, 4);
    if (!data.empty()) {
        output.write(reinterpret_cast<const char*>(data.data()), static_cast<std::streamsize>(data.size()));
    }
    write_u32_be(output, crc32(type, data));
}

void write_png(const std::string& path, const Image& image) {
    ensure_parent_dir(path);
    std::ofstream output(path, std::ios::binary);
    if (!output) {
        throw std::runtime_error("cannot open output image: " + path);
    }
    const std::array<std::uint8_t, 8> signature{{137, 80, 78, 71, 13, 10, 26, 10}};
    output.write(reinterpret_cast<const char*>(signature.data()), static_cast<std::streamsize>(signature.size()));

    std::vector<std::uint8_t> ihdr;
    ihdr.reserve(13);
    append_u32_be(ihdr, static_cast<std::uint32_t>(image.width));
    append_u32_be(ihdr, static_cast<std::uint32_t>(image.height));
    ihdr.push_back(8);  // 8-bit channel depth.
    ihdr.push_back(2);  // truecolor RGB.
    ihdr.push_back(0);  // deflate compression.
    ihdr.push_back(0);  // adaptive filtering.
    ihdr.push_back(0);  // no interlace.
    write_chunk(output, "IHDR", ihdr);
    write_chunk(output, "IDAT", zlib_store(png_scanlines(image)));
    write_chunk(output, "IEND", {});
}

bool has_png_signature(const std::string& path) {
    std::ifstream input(path, std::ios::binary);
    if (!input) {
        return false;
    }
    std::array<unsigned char, 8> header{};
    input.read(reinterpret_cast<char*>(header.data()), static_cast<std::streamsize>(header.size()));
    return input.gcount() == static_cast<std::streamsize>(header.size()) &&
           header == std::array<unsigned char, 8>{{137, 80, 78, 71, 13, 10, 26, 10}};
}

} // namespace lfdraw
