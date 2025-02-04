#!/bin/bash

ISO_URL="ubuntu-24.04.1-live-server-amd64.iso"
OUTPUT_ISO="ubuntu-24.04.1-autoinstall.iso"
AUTOINSTALL_YAML="http/user-data"
CIDATA_DIR="nocloud"
EXTRACTED_ISO="extracted_iso"

# Ensure required tools are installed
if ! command -v xorriso &> /dev/null; then
    echo "xorriso is not installed. Please install it."
    exit 1
fi

if ! command -v grub-mkstandalone &> /dev/null; then
    echo "grub-mkstandalone is not installed. Please install it."
    exit 1
fi

# Extract the ISO with xorriso
mkdir -p "$EXTRACTED_ISO"
xorriso -osirrox on -indev "$ISO_URL" -extract / "$EXTRACTED_ISO"

# Create GRUB BIOS and UEFI images
mkdir -p "$EXTRACTED_ISO/boot/grub/i386-pc"
grub-mkstandalone \
    --format=i386-pc \
    --output="$EXTRACTED_ISO/boot/grub/i386-pc/eltorito.img" \
    --install-modules="linux normal iso9660 biosdisk search search_label search_fs_file" \
    "boot/grub/grub.cfg=$EXTRACTED_ISO/boot/grub/grub.cfg"

mkdir -p "$EXTRACTED_ISO/EFI/boot"
grub-mkstandalone \
    --format=x86_64-efi \
    --output="$EXTRACTED_ISO/EFI/boot/bootx64.efi" \
    --modules="part_gpt part_msdos fat iso9660 linux configfile normal search search_label search_fs_file" \
    "boot/grub/grub.cfg=$EXTRACTED_ISO/boot/grub/grub.cfg"

# Create the cidata directory and user-data file
mkdir -p "$EXTRACTED_ISO/$CIDATA_DIR"
touch "$EXTRACTED_ISO/$CIDATA_DIR/meta-data" # meta-data is required even if empty
cp "$AUTOINSTALL_YAML" "$EXTRACTED_ISO/$CIDATA_DIR/user-data"

# Create grub.cfg
mkdir -p "$EXTRACTED_ISO/boot/grub"
cat << EOF > "$EXTRACTED_ISO/boot/grub/grub.cfg"
set default="0"
set timeout=5

menuentry "Ubuntu Autoinstall" {
    linux /casper/vmlinuz quiet nomodeset net.ifnames=0 biosdevname=0 ip=dhcp autoinstall ds=nocloud\\;s=/cdrom/nocloud ---

    initrd /casper/initrd
}
EOF

xorriso -as mkisofs \
    -o "$OUTPUT_ISO" \
    -isohybrid-gpt-basdat \
    -partition_cyl_align off \
    -partition_offset 16 \
    -eltorito-boot boot/grub/i386-pc/eltorito.img \
    -no-emul-boot \
    -boot-load-size 4 \
    -boot-info-table \
    -eltorito-alt-boot \
    -e EFI/boot/bootx64.efi \
    -no-emul-boot \
    "$EXTRACTED_ISO"


# Clean up
rm -rf "$CIDATA_DIR" "$CIDATA_ISO" "$EXTRACTED_ISO"
echo "Modified ISO created: $OUTPUT_ISO"
sha256sum $OUTPUT_ISO
