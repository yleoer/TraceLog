use base64::{engine::general_purpose::STANDARD, Engine};
use minisign_verify::{PublicKey, Signature};

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let mut args = std::env::args_os().skip(1);
    let installer = args.next().ok_or("缺少安装器路径")?;
    let signature = args.next().ok_or("缺少 .sig 路径")?;
    let config: serde_json::Value = serde_json::from_str(include_str!("../tauri.conf.json"))?;
    let encoded_key = config["plugins"]["updater"]["pubkey"]
        .as_str()
        .ok_or("缺少内置升级公钥")?;
    let public_key = PublicKey::decode(&String::from_utf8(STANDARD.decode(encoded_key)?)?)?;
    let encoded_signature = std::fs::read_to_string(signature)?;
    let signature = Signature::decode(&String::from_utf8(
        STANDARD.decode(encoded_signature.trim())?,
    )?)?;
    public_key.verify(&std::fs::read(installer)?, &signature, true)?;
    println!("安装器签名与应用内置升级公钥匹配");
    Ok(())
}
