using System.Buffers.Binary;

namespace ImageResize.Services;

public enum ImageFormatKind
{
    Jpeg,
    Png,
    Webp,
    Heif,
    Unknown,
}

/// <summary>图片格式识别（按文件头魔数），用于保存时还原原格式。</summary>
public static class ImageFormat
{
    private static readonly HashSet<string> HeifBrands = new(StringComparer.Ordinal)
    {
        "heic", "heix", "hevc", "hevx", "heim", "heis", "hevm", "hevs", "mif1", "msf1",
    };

    /// <summary>按文件头魔数判断格式。</summary>
    public static ImageFormatKind Sniff(ReadOnlySpan<byte> header)
    {
        if (header.Length >= 3 && header[0] == 0xFF && header[1] == 0xD8 && header[2] == 0xFF)
            return ImageFormatKind.Jpeg;
        if (header.Length >= 8 &&
            header[0] == 0x89 && header[1] == 0x50 && header[2] == 0x4E && header[3] == 0x47 &&
            header[4] == 0x0D && header[5] == 0x0A && header[6] == 0x1A && header[7] == 0x0A)
            return ImageFormatKind.Png;
        if (header.Length >= 12 &&
            header[0] == 0x52 && header[1] == 0x49 && header[2] == 0x46 && header[3] == 0x46 &&
            header[8] == 0x57 && header[9] == 0x45 && header[10] == 0x42 && header[11] == 0x50)
            return ImageFormatKind.Webp;
        if (header.Length >= 12 &&
            header[4] == 0x66 && header[5] == 0x74 && header[6] == 0x79 && header[7] == 0x70)
        {
            var brand = System.Text.Encoding.ASCII.GetString(header.Slice(8, 4));
            return HeifBrands.Contains(brand) ? ImageFormatKind.Heif : ImageFormatKind.Unknown;
        }
        return ImageFormatKind.Unknown;
    }

    /// <summary>
    /// WebP 是否为无损：按 RIFF 结构逐个 chunk 遍历，用第一个图像数据块（VP8L / VP8）判定。
    /// 不能只看固定偏移——ICCP / EXIF / XMP 等元数据块的长度不固定，会把 VP8L 推到任意位置。
    /// </summary>
    public static bool IsWebpLossless(string path)
    {
        try
        {
            using var fs = File.OpenRead(path);
            long length = fs.Length;

            var head = new byte[12];
            if (fs.ReadAtLeast(head, head.Length, throwOnEndOfStream: false) < head.Length) return false;
            if (!(head[0] == (byte)'R' && head[1] == (byte)'I' && head[2] == (byte)'F' && head[3] == (byte)'F' &&
                  head[8] == (byte)'W' && head[9] == (byte)'E' && head[10] == (byte)'B' && head[11] == (byte)'P'))
            {
                return false;
            }

            var chunk = new byte[8];
            while (fs.Position + chunk.Length <= length)
            {
                if (fs.ReadAtLeast(chunk, chunk.Length, throwOnEndOfStream: false) < chunk.Length) return false;
                var fourcc = new ReadOnlySpan<byte>(chunk, 0, 4);
                long size = BinaryPrimitives.ReadUInt32LittleEndian(chunk.AsSpan(4, 4));

                if (fourcc.SequenceEqual("VP8L"u8)) return true;   // 无损图像数据
                if (fourcc.SequenceEqual("VP8 "u8)) return false;  // 有损图像数据

                if (fourcc.SequenceEqual("ANMF"u8))
                {
                    // 动图：帧内前 16 字节是帧头，之后才是该帧的图像数据块
                    var frame = new byte[20];
                    if (fs.ReadAtLeast(frame, frame.Length, throwOnEndOfStream: false) < frame.Length) return false;
                    return frame.AsSpan(16, 4).SequenceEqual("VP8L"u8);
                }

                // 其余是元数据/容器块（VP8X / ICCP / EXIF / XMP / ALPH / ANIM …），整块跳过；RIFF 块按偶数字节对齐
                long skip = size + (size & 1);
                if (fs.Position + skip > length) return false;
                fs.Seek(skip, SeekOrigin.Current);
            }

            return false;
        }
        catch (IOException)
        {
            // 读失败时按有损处理：保存走有损编码，结果不会比误当无损更糟
            return false;
        }
    }

    /// <summary>探测文件的格式：先魔数嗅探，UNKNOWN 时按扩展名兜底。返回格式与读取到的文件头。</summary>
    public static (ImageFormatKind Kind, byte[] Header) Detect(string path)
    {
        var header = ReadHeader(path, 48);
        var kind = Sniff(header);
        if (kind != ImageFormatKind.Unknown) return (kind, header);

        kind = Path.GetExtension(path).ToLowerInvariant() switch
        {
            ".jpg" or ".jpeg" => ImageFormatKind.Jpeg,
            ".png" => ImageFormatKind.Png,
            ".webp" => ImageFormatKind.Webp,
            ".heic" or ".heif" => ImageFormatKind.Heif,
            _ => ImageFormatKind.Unknown,
        };
        return (kind, header);
    }

    private static byte[] ReadHeader(string path, int count)
    {
        using var fs = File.OpenRead(path);
        var buf = new byte[count];
        var off = 0;
        while (off < count)
        {
            var n = fs.Read(buf, off, count - off);
            if (n <= 0) break;
            off += n;
        }
        return off == buf.Length ? buf : buf[..off];
    }
}
