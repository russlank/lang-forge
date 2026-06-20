using System.Buffers.Binary;
using System.IO.Compression;
using System.Text;

namespace LangForge.Examples.Draw;

/// <summary>Minimal PNG encoder for RGB images.</summary>
internal static class PngWriter
{
    private static readonly byte[] Signature = [137, 80, 78, 71, 13, 10, 26, 10];

    /// <summary>Writes the image as a PNG file.</summary>
    public static void Write(string path, ImageBuffer image)
    {
        using var file = File.Create(path);
        file.Write(Signature);
        WriteChunk(file, "IHDR", BuildHeader(image.Width, image.Height));
        WriteChunk(file, "IDAT", CompressScanlines(image));
        WriteChunk(file, "IEND", []);
    }

    private static byte[] BuildHeader(int width, int height)
    {
        var data = new byte[13];
        BinaryPrimitives.WriteInt32BigEndian(data.AsSpan(0, 4), width);
        BinaryPrimitives.WriteInt32BigEndian(data.AsSpan(4, 4), height);
        data[8] = 8;  // 8-bit channel depth
        data[9] = 2;  // truecolor RGB
        data[10] = 0; // deflate compression
        data[11] = 0; // adaptive filtering
        data[12] = 0; // no interlace
        return data;
    }

    private static byte[] CompressScanlines(ImageBuffer image)
    {
        using var raw = new MemoryStream();
        for (var y = 0; y < image.Height; y++)
        {
            raw.WriteByte(0); // PNG filter type 0: none
            for (var x = 0; x < image.Width; x++)
            {
                var pixel = image.GetPixel(x, y);
                raw.WriteByte(pixel.R);
                raw.WriteByte(pixel.G);
                raw.WriteByte(pixel.B);
            }
        }
        using var compressed = new MemoryStream();
        using (var zlib = new ZLibStream(compressed, CompressionLevel.Fastest, leaveOpen: true))
        {
            raw.Position = 0;
            raw.CopyTo(zlib);
        }
        return compressed.ToArray();
    }

    private static void WriteChunk(Stream stream, string type, byte[] data)
    {
        Span<byte> length = stackalloc byte[4];
        BinaryPrimitives.WriteInt32BigEndian(length, data.Length);
        stream.Write(length);
        var typeBytes = Encoding.ASCII.GetBytes(type);
        stream.Write(typeBytes);
        stream.Write(data);
        var crc = Crc32(typeBytes, data);
        Span<byte> crcBytes = stackalloc byte[4];
        BinaryPrimitives.WriteUInt32BigEndian(crcBytes, crc);
        stream.Write(crcBytes);
    }

    private static uint Crc32(byte[] type, byte[] data)
    {
        var crc = 0xffffffffu;
        foreach (var b in type)
        {
            crc = UpdateCrc(crc, b);
        }
        foreach (var b in data)
        {
            crc = UpdateCrc(crc, b);
        }
        return crc ^ 0xffffffffu;
    }

    private static uint UpdateCrc(uint crc, byte b)
    {
        crc ^= b;
        for (var i = 0; i < 8; i++)
        {
            crc = (crc & 1) == 1 ? 0xedb88320u ^ (crc >> 1) : crc >> 1;
        }
        return crc;
    }
}
