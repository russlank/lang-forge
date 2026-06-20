namespace LangForge.Examples.Draw;

/// <summary>In-memory RGB image used by the renderer and PNG writer.</summary>
internal sealed class ImageBuffer
{
    private readonly ColorRgb[] _pixels;

    /// <summary>Creates a white image with the requested size.</summary>
    public ImageBuffer(int width, int height)
    {
        Width = width;
        Height = height;
        _pixels = Enumerable.Repeat(new ColorRgb(255, 255, 255), width * height).ToArray();
    }

    /// <summary>Image width in pixels.</summary>
    public int Width { get; }

    /// <summary>Image height in pixels.</summary>
    public int Height { get; }

    /// <summary>Sets one pixel if the coordinates are inside the image.</summary>
    public void SetPixel(int x, int y, ColorRgb color)
    {
        if (x < 0 || y < 0 || x >= Width || y >= Height)
        {
            return;
        }
        _pixels[y * Width + x] = color;
    }

    /// <summary>Returns one pixel without bounds checks.</summary>
    public ColorRgb GetPixel(int x, int y) => _pixels[y * Width + x];

    /// <summary>Fills every pixel with a color.</summary>
    public void Fill(ColorRgb color)
    {
        Array.Fill(_pixels, color);
    }
}
