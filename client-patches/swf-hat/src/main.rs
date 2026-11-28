//! Adds a hat made of a round photo to a hats SWF file of the 1.61.7 client:
//! `swf-hat <in.swf> <out.swf> <photo.png> <ClassName> <cx> <cy> <diameter>`.
//!
//! The client draws a hat from the class its hat table names (`iconClassName`), which it finds in the hats
//! files it loads (data/common/objects/hats*.swf). The tool adds to the file:
//! - the photo, cut to a circle with a thin dark rim, as a lossless bitmap with alpha (DefineBitsLossless2);
//! - a square shape filled with it (DefineShape3), `diameter` hat pixels wide and centred on (`cx`, `cy`)
//!   (hats are drawn ten times their size: the hat table scales them by 0.1);
//! - a sprite with the shape (DefineSprite);
//! - a class `ClassName` extending flash.display.MovieClip (DoABC2), and the sprite linked to it
//!   (SymbolClass).
//!
//! The tags go before the file's last ShowFrame, after its own classes. The output keeps the input's
//! compression (FWS or CWS).

use std::io::{Read, Write};
use std::{env, fs, process};

use flate2::{read::ZlibDecoder, write::ZlibEncoder, Compression};

// ---- Bits ----------------------------------------------------------------------------------------------

struct Bits<'a> {
    data: &'a [u8],
    pos: usize,
}

impl Bits<'_> {
    fn read(&mut self, n: usize) -> u32 {
        let mut v = 0;
        for _ in 0..n {
            let bit = (self.data[self.pos / 8] >> (7 - self.pos % 8)) & 1;
            v = v << 1 | u32::from(bit);
            self.pos += 1;
        }
        v
    }
}

#[derive(Default)]
struct BitWriter {
    out: Vec<u8>,
    pos: usize,
}

impl BitWriter {
    fn put(&mut self, v: u32, n: usize) {
        for i in (0..n).rev() {
            if self.pos % 8 == 0 {
                self.out.push(0);
            }
            let bit = ((v >> i) & 1) as u8;
            *self.out.last_mut().unwrap() |= bit << (7 - self.pos % 8);
            self.pos += 1;
        }
    }

    fn signed(&mut self, v: i32, n: usize) {
        self.put(v as u32 & ((1u64 << n) - 1) as u32, n);
    }

    /// Ends the bit field: the next field starts on a byte.
    fn align(&mut self) {
        self.pos = self.out.len() * 8;
    }
}

/// The bits a signed number takes, with its sign bit.
fn signed_bits(v: i32) -> usize {
    (33 - v.unsigned_abs().leading_zeros() as usize).max(1)
}

fn rect(w: &mut BitWriter, r: [i32; 4]) {
    let n = r.iter().map(|&v| signed_bits(v)).max().unwrap();
    w.put(n as u32, 5);
    for v in r {
        w.signed(v, n);
    }
    w.align();
}

// ---- Tags ----------------------------------------------------------------------------------------------

const TAG_END: u16 = 0;
const TAG_SHOW_FRAME: u16 = 1;
const TAG_PLACE_OBJECT2: u16 = 26;
const TAG_DEFINE_SHAPE3: u16 = 32;
const TAG_DEFINE_BITS_LOSSLESS2: u16 = 36;
const TAG_DEFINE_SPRITE: u16 = 39;
const TAG_SYMBOL_CLASS: u16 = 76;
const TAG_DO_ABC2: u16 = 82;

/// The tags whose body starts with the id of the character they define.
const DEFINE_TAGS: &[u16] = &[
    2, 6, 10, 11, 20, 21, 22, 32, 33, 34, 35, 36, 37, 39, 46, 48, 60, 75, 83, 84, 87, 90, 91,
];

fn tag(code: u16, body: &[u8]) -> Vec<u8> {
    let mut out = Vec::with_capacity(body.len() + 6);
    if body.len() < 63 && code != TAG_DEFINE_BITS_LOSSLESS2 {
        out.extend_from_slice(&(code << 6 | body.len() as u16).to_le_bytes());
    } else {
        out.extend_from_slice(&(code << 6 | 0x3f).to_le_bytes());
        out.extend_from_slice(&(body.len() as u32).to_le_bytes());
    }
    out.extend_from_slice(body);
    out
}

/// A tag of the file: its code and where it starts and ends (header included).
struct Tag {
    code: u16,
    start: usize,
    end: usize,
}

/// The tags after the header's frame size, frame rate and frame count, and where they start.
fn tags(body: &[u8]) -> (usize, Vec<Tag>) {
    let mut bits = Bits { data: body, pos: 0 };
    let nbits = bits.read(5) as usize;
    let first = (5 + 4 * nbits).div_ceil(8) + 4;
    let mut out = Vec::new();
    let mut pos = first;
    while pos + 2 <= body.len() {
        let h = u16::from_le_bytes([body[pos], body[pos + 1]]);
        let (code, mut len, mut hdr) = (h >> 6, usize::from(h & 0x3f), 2);
        if len == 0x3f {
            len = u32::from_le_bytes(body[pos + 2..pos + 6].try_into().unwrap()) as usize;
            hdr = 6;
        }
        out.push(Tag { code, start: pos, end: pos + hdr + len });
        pos += hdr + len;
        if code == TAG_END {
            break;
        }
    }
    (first, out)
}

fn body_of<'a>(body: &'a [u8], t: &Tag) -> &'a [u8] {
    let hdr = if u16::from_le_bytes([body[t.start], body[t.start + 1]]) & 0x3f == 0x3f { 6 } else { 2 };
    &body[t.start + hdr..t.end]
}

// ---- The photo -----------------------------------------------------------------------------------------

/// The photo's centred square, cut to a circle with a dark rim: its side and its premultiplied ARGB pixels.
fn round_photo(png_data: &[u8]) -> (u16, Vec<u8>) {
    let mut dec = png::Decoder::new(png_data);
    dec.set_transformations(png::Transformations::normalize_to_color8());
    let mut reader = dec.read_info().expect("a PNG file");
    let mut buf = vec![0; reader.output_buffer_size()];
    let info = reader.next_frame(&mut buf).expect("PNG data");
    let (w, h) = (info.width as usize, info.height as usize);
    let channels = match info.color_type {
        png::ColorType::Rgba => 4,
        png::ColorType::Rgb => 3,
        png::ColorType::GrayscaleAlpha => 2,
        png::ColorType::Grayscale => 1,
        png::ColorType::Indexed => unreachable!("expanded by normalize_to_color8"),
    };
    let side = w.min(h).min(1024);
    let (x0, y0) = ((w - side) / 2, (h - side) / 2);
    let r = side as f64 / 2.0;
    let rim = (side as f64 * 0.03).max(2.0);
    let mut out = Vec::with_capacity(side * side * 4);
    for y in 0..side {
        for x in 0..side {
            let i = ((y0 + y) * w + x0 + x) * channels;
            let px = &buf[i..i + channels];
            let (mut cr, mut cg, mut cb, a0) = match channels {
                4 => (px[0], px[1], px[2], px[3]),
                3 => (px[0], px[1], px[2], 255),
                2 => (px[0], px[0], px[0], px[1]),
                _ => (px[0], px[0], px[0], 255),
            };
            let d = ((x as f64 + 0.5 - r).powi(2) + (y as f64 + 0.5 - r).powi(2)).sqrt();
            // 1 inside, 0 outside, smooth over the edge pixel.
            let inside = (r - d).clamp(0.0, 1.0);
            if d > r - rim {
                (cr, cg, cb) = (40, 30, 25);
            }
            let a = (f64::from(a0) * inside).round() as u32;
            let pm = |c: u8| (u32::from(c) * a / 255) as u8;
            out.extend_from_slice(&[a as u8, pm(cr), pm(cg), pm(cb)]);
        }
    }
    (side as u16, out)
}

// ---- The class -----------------------------------------------------------------------------------------

fn u30(out: &mut Vec<u8>, mut v: u32) {
    loop {
        let b = (v & 0x7f) as u8;
        v >>= 7;
        if v == 0 {
            out.push(b);
            return;
        }
        out.push(b | 0x80);
    }
}

fn abc_string(out: &mut Vec<u8>, s: &str) {
    u30(out, s.len() as u32);
    out.extend_from_slice(s.as_bytes());
}

/// An ABC file with `public class <name> extends flash.display.MovieClip` in the top package, its
/// constructor calling MovieClip's.
fn movie_clip_class(name: &str) -> Vec<u8> {
    let mut a = Vec::new();
    a.extend_from_slice(&16u16.to_le_bytes()); // minor version
    a.extend_from_slice(&46u16.to_le_bytes()); // major version
    u30(&mut a, 0); // ints
    u30(&mut a, 0); // uints
    u30(&mut a, 0); // doubles
    // Strings: 1 "", 2 the class, 3 "flash.display", 4 "MovieClip".
    u30(&mut a, 5);
    for s in ["", name, "flash.display", "MovieClip"] {
        abc_string(&mut a, s);
    }
    // Namespaces: 1 the top package, 2 flash.display.
    u30(&mut a, 3);
    a.extend_from_slice(&[0x16, 1, 0x16, 3]);
    u30(&mut a, 0); // namespace sets
    // Multinames: 1 the class, 2 flash.display::MovieClip.
    u30(&mut a, 3);
    a.extend_from_slice(&[0x07, 1, 2, 0x07, 2, 4]);
    // Methods: 0 the constructor, 1 the class initializer, 2 the script initializer; no parameters.
    u30(&mut a, 3);
    for _ in 0..3 {
        a.extend_from_slice(&[0, 0, 0, 0]);
    }
    u30(&mut a, 0); // metadata
    u30(&mut a, 1); // classes
    a.extend_from_slice(&[1, 2, 0x01, 0, 0, 0]); // name, super, sealed, no interfaces, iinit 0, no traits
    a.extend_from_slice(&[1, 0]); // cinit 1, no traits
    u30(&mut a, 1); // scripts
    a.extend_from_slice(&[2, 1, 1, 0x04, 1, 0]); // init 2; one trait: the class (slot 1, class 0)
    u30(&mut a, 3); // method bodies
    let bodies: [(u8, u8, u8, u8, &[u8]); 3] = [
        // getlocal0 pushscope getlocal0 constructsuper(0) returnvoid
        (0, 1, 1, 2, &[0xd0, 0x30, 0xd0, 0x49, 0x00, 0x47]),
        // getlocal0 pushscope returnvoid
        (1, 1, 1, 2, &[0xd0, 0x30, 0x47]),
        // getlocal0 pushscope getscopeobject(0) getlex(MovieClip) newclass(0) initproperty(class) returnvoid
        (2, 2, 0, 1, &[0xd0, 0x30, 0x65, 0x00, 0x60, 0x02, 0x58, 0x00, 0x68, 0x01, 0x47]),
    ];
    for (method, max_stack, init_scope, max_scope, code) in bodies {
        a.extend_from_slice(&[method, max_stack, 1, init_scope, max_scope]);
        u30(&mut a, code.len() as u32);
        a.extend_from_slice(code);
        a.extend_from_slice(&[0, 0]); // no exceptions, no traits
    }
    a
}

// ---- The hat -------------------------------------------------------------------------------------------

/// The tags of the hat: bitmap `ids.0`, shape `ids.1`, sprite `ids.2`, the class and its link.
fn hat_tags(ids: (u16, u16, u16), class: &str, side: u16, argb: &[u8], cx: i32, cy: i32, diameter: i32) -> Vec<u8> {
    let mut out = Vec::new();

    let mut z = ZlibEncoder::new(Vec::new(), Compression::best());
    z.write_all(argb).unwrap();
    let mut bits = Vec::new();
    bits.extend_from_slice(&ids.0.to_le_bytes());
    bits.push(5); // 32-bit ARGB
    bits.extend_from_slice(&side.to_le_bytes());
    bits.extend_from_slice(&side.to_le_bytes());
    bits.extend(z.finish().unwrap());
    out.extend(tag(TAG_DEFINE_BITS_LOSSLESS2, &bits));

    // In twips (1/20 px).
    let d = diameter * 20;
    let (x0, y0) = (cx * 20 - d / 2, cy * 20 - d / 2);
    let mut s = BitWriter::default();
    s.out.extend_from_slice(&ids.1.to_le_bytes());
    s.pos = s.out.len() * 8;
    rect(&mut s, [x0, x0 + d, y0, y0 + d]);
    s.out.extend_from_slice(&[1, 0x41]); // one fill: a clipped, smoothed bitmap
    s.out.extend_from_slice(&ids.0.to_le_bytes());
    s.pos = s.out.len() * 8;
    // Its matrix: a bitmap pixel is d/side twips, from the square's corner (16.16 fixed point scale).
    let scale = ((f64::from(d) / f64::from(side)) * 65536.0).round() as i32;
    s.put(1, 1);
    let n = signed_bits(scale);
    s.put(n as u32, 5);
    s.signed(scale, n);
    s.signed(scale, n);
    s.put(0, 1); // no rotation
    let n = signed_bits(x0).max(signed_bits(y0));
    s.put(n as u32, 5);
    s.signed(x0, n);
    s.signed(y0, n);
    s.align();
    s.out.push(0); // no line styles
    s.out.push(0x10); // 1 fill bit, 0 line bits
    s.pos = s.out.len() * 8;
    // Move to the corner with fill 1 (the right side of a clockwise outline), then the four edges.
    s.put(0, 1);
    s.put(0b00101, 5);
    let n = signed_bits(x0).max(signed_bits(y0));
    s.put(n as u32, 5);
    s.signed(x0, n);
    s.signed(y0, n);
    s.put(1, 1);
    let n = signed_bits(d).max(signed_bits(-d));
    for (vertical, delta) in [(false, d), (true, d), (false, -d), (true, -d)] {
        s.put(0b11, 2); // an edge, straight
        s.put((n - 2) as u32, 4);
        s.put(0, 1); // not a general line
        s.put(u32::from(vertical), 1);
        s.signed(delta, n);
    }
    s.put(0, 6); // end of the shape
    s.align();
    out.extend(tag(TAG_DEFINE_SHAPE3, &s.out));

    let mut sp = Vec::new();
    sp.extend_from_slice(&ids.2.to_le_bytes());
    sp.extend_from_slice(&1u16.to_le_bytes()); // one frame
    let mut place = vec![0x02]; // has a character
    place.extend_from_slice(&1u16.to_le_bytes()); // depth
    place.extend_from_slice(&ids.1.to_le_bytes());
    sp.extend(tag(TAG_PLACE_OBJECT2, &place));
    sp.extend(tag(TAG_SHOW_FRAME, &[]));
    sp.extend(tag(TAG_END, &[]));
    out.extend(tag(TAG_DEFINE_SPRITE, &sp));

    let mut abc = 1u32.to_le_bytes().to_vec(); // lazy initialization
    abc.extend_from_slice(class.as_bytes());
    abc.push(0);
    abc.extend(movie_clip_class(class));
    out.extend(tag(TAG_DO_ABC2, &abc));

    let mut sym = 1u16.to_le_bytes().to_vec();
    sym.extend_from_slice(&ids.2.to_le_bytes());
    sym.extend_from_slice(class.as_bytes());
    sym.push(0);
    out.extend(tag(TAG_SYMBOL_CLASS, &sym));
    out
}

/// The SWF body (after the 8-byte header) with the hat added.
fn add_hat(body: &[u8], class: &str, side: u16, argb: &[u8], cx: i32, cy: i32, diameter: i32) -> Result<Vec<u8>, String> {
    let (_, list) = tags(body);
    let mut max_id = 0u16;
    for t in &list {
        let b = body_of(body, t);
        if DEFINE_TAGS.contains(&t.code) && b.len() >= 2 {
            max_id = max_id.max(u16::from_le_bytes([b[0], b[1]]));
        }
        if t.code == TAG_SYMBOL_CLASS {
            let mut p = 2;
            for _ in 0..u16::from_le_bytes([b[0], b[1]]) {
                let end = p + 2 + b[p + 2..].iter().position(|&c| c == 0).unwrap_or(0);
                if &b[p + 2..end] == class.as_bytes() {
                    return Err(format!("the file has a class {class} already"));
                }
                p = end + 1;
            }
        }
    }
    let at = list
        .iter()
        .rev()
        .find(|t| t.code == TAG_SHOW_FRAME)
        .ok_or("no ShowFrame in the file")?
        .start;
    if max_id > u16::MAX - 3 {
        return Err("no character ids left".into());
    }
    let ids = (max_id + 1, max_id + 2, max_id + 3);
    let mut out = body[..at].to_vec();
    out.extend(hat_tags(ids, class, side, argb, cx, cy, diameter));
    out.extend_from_slice(&body[at..]);
    Ok(out)
}

fn fail(msg: impl std::fmt::Display) -> ! {
    eprintln!("{msg}");
    process::exit(1);
}

fn main() {
    let args: Vec<String> = env::args().collect();
    if args.len() != 8 {
        eprintln!("usage: swf-hat <in.swf> <out.swf> <photo.png> <ClassName> <cx> <cy> <diameter>");
        process::exit(2);
    }
    let class = &args[4];
    if class.is_empty() || !class.chars().all(|c| c.is_ascii_alphanumeric() || c == '_') || class.as_bytes()[0].is_ascii_digit() {
        fail("ClassName: letters, digits and _, not starting with a digit");
    }
    let num = |s: &String| s.parse::<i32>().unwrap_or_else(|_| fail(format!("{s}: not a whole number")));
    let (cx, cy, diameter) = (num(&args[5]), num(&args[6]), num(&args[7]));
    if !(1..=2000).contains(&diameter) || cx.abs() > 5000 || cy.abs() > 5000 {
        fail("diameter: 1 to 2000 hat pixels; cx, cy: within 5000");
    }
    let file = fs::read(&args[1]).unwrap_or_else(|e| fail(format!("{}: {e}", args[1])));
    if file.len() < 8 || !matches!(&file[..3], b"FWS" | b"CWS") {
        fail(format!("{}: not an FWS or CWS file", args[1]));
    }
    let photo = fs::read(&args[3]).unwrap_or_else(|e| fail(format!("{}: {e}", args[3])));
    let (side, argb) = round_photo(&photo);
    let compressed = &file[..3] == b"CWS";
    let body = if compressed {
        let mut b = Vec::new();
        ZlibDecoder::new(&file[8..]).read_to_end(&mut b).expect("zlib data");
        b
    } else {
        file[8..].to_vec()
    };
    let body = add_hat(&body, class, side, &argb, cx, cy, diameter).unwrap_or_else(|e| fail(e));
    let mut out = file[..4].to_vec();
    out.extend_from_slice(&(body.len() as u32 + 8).to_le_bytes());
    if compressed {
        let mut z = ZlibEncoder::new(Vec::new(), Compression::best());
        z.write_all(&body).unwrap();
        out.extend(z.finish().unwrap());
    } else {
        out.extend(body);
    }
    fs::write(&args[2], out).unwrap_or_else(|e| fail(format!("{}: {e}", args[2])));
    println!("{}: {class}, a {side}px photo, {diameter} hat pixels at ({cx}, {cy})", args[2]);
}

#[cfg(test)]
mod tests {
    use super::*;

    /// A 100x100 SWF body: a shape (id 7), a ShowFrame and the End.
    fn swf() -> Vec<u8> {
        let mut w = BitWriter::default();
        rect(&mut w, [0, 2000, 0, 2000]);
        let mut b = w.out;
        b.extend_from_slice(&[0, 30, 1, 0]);
        b.extend(tag(2, &[7, 0, 1, 2, 3]));
        b.extend(tag(TAG_SHOW_FRAME, &[]));
        b.extend(tag(TAG_END, &[]));
        b
    }

    #[test]
    fn adds_the_hat_before_the_last_frame() {
        let argb = vec![255; 4 * 4 * 4];
        let out = add_hat(&swf(), "XeroxHat", 4, &argb, 160, 190, 300).unwrap();
        let (_, list) = tags(&out);
        let codes: Vec<u16> = list.iter().map(|t| t.code).collect();
        assert_eq!(
            codes,
            [2, TAG_DEFINE_BITS_LOSSLESS2, TAG_DEFINE_SHAPE3, TAG_DEFINE_SPRITE, TAG_DO_ABC2, TAG_SYMBOL_CLASS, TAG_SHOW_FRAME, TAG_END]
        );
        assert_eq!(&body_of(&out, &list[1])[..2], &[8, 0]);
        assert_eq!(&body_of(&out, &list[3])[..2], &[10, 0]);
        assert_eq!(body_of(&out, &list[5]), b"\x01\x00\x0a\x00XeroxHat\x00");
        assert!(add_hat(&out, "XeroxHat", 4, &argb, 0, 0, 10).is_err(), "a second XeroxHat");
    }

    #[test]
    fn writes_u30() {
        let mut v = Vec::new();
        u30(&mut v, 300);
        assert_eq!(v, [0xac, 0x02]);
    }
}
