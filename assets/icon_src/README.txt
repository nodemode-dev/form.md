form.md app icon: the owner's #F edit (form-md_edit_source.png) is the source.
art1024.png is that tile upscaled 4x with EDSR (opencv dnn_superres), lightly
denoised and sharpened. The old pointer and tag were inpainted out and drawn
again as vectors at the same spots; the tile edge was rebuilt from the page
mask. Every icon file is a resize of that 1024 image (the macOS one as an
824 px tile with a 100 px margin and drop shadow).

Since v0.45.2 the icon files are made by make_icons.py (Pillow and NumPy,
run from the repository root). The first cut-out followed the mockup's
light page and its grey shadow, which left a light rim round every icon and
a grey crescent at the lower right, invisible on a white background. The
script cuts the tile with a superellipse (n 5.4, half-width 506.9 px)
fitted 2 px inside the tile's dark edge, drawn at 4x for a smooth edge,
fills the transparent pixels with the tile's edge colour and resizes with
premultiplied alpha, so no light pixel reaches any size.

Since v0.45.3 the artwork comes from approved1024.png, the icon the owner
chose in v0.45.1 (the white ".MD" in a white outline beside the pointer).
art1024.png still carries the earlier tag, a white pill with dark text, and
v0.45.2 was wrongly built from it alone. Every pixel approved1024.png holds
opaque, 4 px in from its old edge, is kept exactly; art1024.png only fills
the outer few pixels the new cut reaches.
