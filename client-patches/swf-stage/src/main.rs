//! Sets the stage size in a SWF file's header: `swf-stage <in.swf> <out.swf> <width> <height>`.
//!
//! The header's frame size is a RECT of four signed numbers in twips (1/20 px), each `nbits` long.
//! The tool rewrites it and keeps the rest of the file. FWS (uncompressed) and CWS (zlib) files are read;
//! the output keeps the input's compression.

use std::io::{Read, Write};
use std::{env, fs, process};

use flate2::{read::ZlibDecoder, write::ZlibEncoder, Compression};

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

fn write_bits(out: &mut Vec<u8>, pos: &mut usize, v: u32, n: usize) {
    for i in (0..n).rev() {
        if *pos % 8 == 0 {
            out.push(0);
        }
        let bit = ((v >> i) & 1) as u8;
        *out.last_mut().unwrap() |= bit << (7 - *pos % 8);
        *pos += 1;
    }
}

/// The bits a signed number takes, with its sign bit.
fn signed_bits(v: i32) -> usize {
    (33 - v.unsigned_abs().leading_zeros() as usize).max(1)
}

/// The body after the 8-byte header with a frame size of `w`x`h` pixels; returns it and the old size.
fn set_stage(body: &[u8], w: i32, h: i32) -> (Vec<u8>, (f64, f64)) {
    let mut bits = Bits { data: body, pos: 0 };
    let nbits = bits.read(5) as usize;
    let old: Vec<i32> = (0..4)
        .map(|_| {
            let v = bits.read(nbits);
            ((v << (32 - nbits)) as i32) >> (32 - nbits)
        })
        .collect();
    let rect_len = bits.pos.div_ceil(8);
    let new = [0, w * 20, 0, h * 20];
    let n = new.iter().map(|&v| signed_bits(v)).max().unwrap();
    let mut out = Vec::with_capacity(body.len() + 4);
    let mut pos = 0;
    write_bits(&mut out, &mut pos, n as u32, 5);
    for v in new {
        write_bits(&mut out, &mut pos, v as u32 & ((1u64 << n) - 1) as u32, n);
    }
    out.extend_from_slice(&body[rect_len..]);
    let size = (f64::from(old[1] - old[0]) / 20.0, f64::from(old[3] - old[2]) / 20.0);
    (out, size)
}

fn main() {
    let args: Vec<String> = env::args().collect();
    if args.len() != 5 {
        eprintln!("usage: swf-stage <in.swf> <out.swf> <width> <height>");
        process::exit(2);
    }
    let (w, h): (i32, i32) = match (args[3].parse(), args[4].parse()) {
        (Ok(w), Ok(h)) if w > 0 && h > 0 && w < 1600 && h < 1600 => (w, h),
        _ => {
            eprintln!("width and height: whole pixels, 1 to 1599");
            process::exit(2);
        }
    };
    let file = fs::read(&args[1]).unwrap_or_else(|e| {
        eprintln!("{}: {e}", args[1]);
        process::exit(1);
    });
    if file.len() < 8 || !matches!(&file[..3], b"FWS" | b"CWS") {
        eprintln!("{}: not an FWS or CWS file", args[1]);
        process::exit(1);
    }
    let compressed = &file[..3] == b"CWS";
    let body = if compressed {
        let mut b = Vec::new();
        ZlibDecoder::new(&file[8..]).read_to_end(&mut b).expect("zlib data");
        b
    } else {
        file[8..].to_vec()
    };
    let (body, (ow, oh)) = set_stage(&body, w, h);
    let mut out = file[..4].to_vec();
    out.extend_from_slice(&(body.len() as u32 + 8).to_le_bytes());
    if compressed {
        let mut z = ZlibEncoder::new(Vec::new(), Compression::best());
        z.write_all(&body).unwrap();
        out.extend(z.finish().unwrap());
    } else {
        out.extend(body);
    }
    fs::write(&args[2], out).unwrap_or_else(|e| {
        eprintln!("{}: {e}", args[2]);
        process::exit(1);
    });
    println!("{}: {ow}x{oh} -> {w}x{h}", args[2]);
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn rewrites_the_frame_size() {
        // 800x600 with 16-bit fields, then the frame rate and count.
        let mut body = Vec::new();
        let mut pos = 0;
        write_bits(&mut body, &mut pos, 16, 5);
        for v in [0, 16000, 0, 12000] {
            write_bits(&mut body, &mut pos, v, 16);
        }
        body.extend_from_slice(&[0, 30, 1, 0]);
        let (out, old) = set_stage(&body, 960, 640);
        assert_eq!(old, (800.0, 600.0));
        let mut bits = Bits { data: &out, pos: 0 };
        let n = bits.read(5) as usize;
        let rect: Vec<u32> = (0..4).map(|_| bits.read(n)).collect();
        assert_eq!(rect, [0, 19200, 0, 12800]);
        assert_eq!(&out[bits.pos.div_ceil(8)..], &[0, 30, 1, 0]);
    }
}
