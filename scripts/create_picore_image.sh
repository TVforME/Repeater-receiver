#!/bin/bash

# Borrow and extended from :-
# https://github.com/4xx/picore-headless-setup/blob/master/picore-headless-setup.sh

trap "exit 1" TERM
export TOP_PID=$$

# ==============================================================================
# IMAGE CONFIGURATION for Repeater-dvb  piCore64
# ==============================================================================

PICORE_APP="dvb"
PICORE_ARCH="aarch64"
PICORE_VERSION="14.x"
PICORE_SUBVERSION="14.1.0"
PICORE_SUBVERSION_SHORT=${PICORE_SUBVERSION%.0*}
PICORE_ROOTFS=rootfs-piCore64-$PICORE_SUBVERSION_SHORT
IMG_BLOCKSIZE=512
IMG_BLOCKS=204800 # 512 * 204800 = 104857600 (~100MB)
#IMG_BLOCKS=409600 # 512 * 409600 = 209715200 (~200MB)


WORK_DIR=./piCore64-$PICORE_SUBVERSION
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
MNT1_DIR="$WORK_DIR/mnt1"
MNT2_DIR="$WORK_DIR/mnt2"
PACKAGES_DIR="$WORK_DIR/pkg"
BOOT_CONFIG="$MNT1_DIR/config.txt"

WGET_OPTS=" -c --no-proxy"

PICORE_BASE_URL="http://www.tinycorelinux.net"
PICORE_REPOSITORY_URL="$PICORE_BASE_URL/$PICORE_VERSION/$PICORE_ARCH"
PICORE_RELEASES_URL="$PICORE_REPOSITORY_URL/releases/RPi"
PICORE_PACKAGES_URL="$PICORE_REPOSITORY_URL/tcz"
PICORE_PACKAGE_EXTENSION="tcz"
PICORE_RELEASE_URL="$PICORE_RELEASES_URL/piCore64-$PICORE_SUBVERSION.zip"
#PICORE_KERNEL_SUFFIX="-$PICORE_KERNEL_VERSION-piCore64"
PICORE_LOCAL_PACKAGE_PATH="tce/optional"
PICORE_LOCAL_MYDATA="tce/mydata"

PICORE_FILESYSTEM_DIR="${WORK_DIR}/filesystem"




PICORE_PACKAGES=(	"file"\
                    "sysfsutils"\
                    "libgpiod"\
					"ncurses"\
					"nano"\
)

PICORE_PACKAGES_DVB=(   "pciutils"\
                        "v4l-dvb-6.1.68-piCore-v8"\
					    "v4l2-utils"\
                        "socat"\
)


##############################################################################

# Define the list of necessary script files to ADD or DELETE files or dirs
SCRIPTS=(
    "add $SCRIPT_DIR/scripts/bootlocal.sh /opt/bootlocal.sh"
    "add $SCRIPT_DIR/scripts/powerbutton.sh /opt/powerbutton.sh"
    "add $SCRIPT_DIR/scripts/powerscript.sh /opt/powerscripts.sh"
    "add $SCRIPT_DIR/scripts/startup.sh /usr/local/etc/init.d/startup.sh"
    "delete /etc/asound.conf"
    "delete /etc/lib/alsa"
    "delete /var/lib/alsa"
    "add $SCRIPT_DIR/scripts/dvb /etc/lib/dvb"
    "modify rng-tools-5.tcz /tce/onboot.lst -d"
    "modify this is a test /etc/lib/dvb/dvb.conf -a"
)


##############################################################################

PICORE_PACKAGES=("${PICORE_PACKAGES[@]}" "${PICORE_PACKAGES_DVB[@]}")
DEPENDENCIES=(	"wget"\
				"md5sum"\
				"unzip"\
				"dd"\
				"losetup"\
				"kpartx"\
				"parted"\
				"e2fsck"\
				"resize2fs"\
				"mount"\
				"umount"\
				"cat"\
				"awk"\
				"tar"
)

function check_script_existence() {
    echo "================================================================================"
    echo " * Checking Script and Directory Existence"
    echo "================================================================================"
    for entry in "${SCRIPTS[@]}"; do
        IFS=' ' read -r action src dst <<< "$entry"
        if [ "$action" == "add" ]; then
            if [ -d "$src" ] || [ -f "$src" ]; then
                echo "  - $src exists."
            else
                echo "Error: $src does not exist. Exiting..."
                exit 1
            fi
        fi
    done
    echo "  - OK"    
    echo ""
}



function manage_scripts_and_directories() {
    echo "================================================================================"
    echo " * Managing Scripts and Directories in mydata"
    echo "================================================================================"
    for entry in "${SCRIPTS[@]}"; do
        IFS=' ' read -r action src dst mod <<< "$entry"
        if [ "$action" == "add" ]; then
            if [ -d "$src" ]; then
                echo "Adding directory $src to $dst"
                sudo mkdir -p "$MNT2_DIR/$PICORE_LOCAL_MYDATA/$dst"
                sudo cp -r "$src/"* "$MNT2_DIR/$PICORE_LOCAL_MYDATA/$dst/"
            elif [ -f "$src" ]; then
                echo "Adding file $src to $dst"
                sudo mkdir -p "$(dirname "$MNT2_DIR/$PICORE_LOCAL_MYDATA/$dst")"
                sudo cp "$src" "$MNT2_DIR/$PICORE_LOCAL_MYDATA/$dst"
            fi
        elif [ "$action" == "delete" ]; then
            if [ -d "$MNT2_DIR/$PICORE_LOCAL_MYDATA/$src" ]; then
                echo "Deleting directory $src"
                sudo rm -rf "$MNT2_DIR/$PICORE_LOCAL_MYDATA/$src"
            elif [ -f "$MNT2_DIR/$PICORE_LOCAL_MYDATA/$src" ]; then
                echo "Deleting file $src"
                sudo rm "$MNT2_DIR/$PICORE_LOCAL_MYDATA/$src"
            fi
        elif [ "$action" == "modify" ]; then
            if [ "$mod" == _"-d" ]; then
                echo "Deleting entry $src from $dst"
                sudo sed -i "/$src/d" "$MNT2_DIR/$PICORE_LOCAL_MYDATA/$dst"
            elif [ "$mod" == _"-a" ]; then
                echo "Adding entry $src to $dst"
                sudo sh -c "echo $src >> $MNT2_DIR/$PICORE_LOCAL_MYDATA/$dst"
            fi
        fi
    done
    echo "  - OK"    
    echo ""
}



function prepare_dirs(){
    echo "================================================================================" 
    echo " * Making directories"
    echo "================================================================================" 
    [ -d $WORK_DIR ] || mkdir $WORK_DIR
    #[ -d $MYDATA_DIR ] || mkdir $MYDATA_DIR
    [ -d $MNT1_DIR ] || mkdir $MNT1_DIR
    [ -d $MNT2_DIR ] || mkdir $MNT2_DIR
    [ -d $PICORE_FILESYSTEM_DIR ] || mkdir $PICORE_FILESYSTEM_DIR
    echo "  - OK"    
    echo ""
}

function command_exists() {
    type "$1" &> /dev/null ;
}

function validate_url(){
if [[ $(wget $WGET_OPTS -S --spider $1 2>&1 | grep 'HTTP/1.1 200 OK') ]]; then return 0; else return 1; fi
}

function check_dependencies(){
    echo "================================================================================" 
	echo " * Checking dependencies"
    echo "================================================================================" 
	for i in "${DEPENDENCIES[@]}"
    	do
    		echo -ne "  - $i"
    		if command_exists $i ; then
    			echo " OK"
    		else
    			echo " ERROR. Please install $i and rerun."
    			kill -s TERM $TOP_PID
    		fi
    done
    echo ""
}

function download_release_maybe(){
    echo "================================================================================" 
    echo " * Downloading PiCore64 Release"
    echo "================================================================================" 
    cd "$SCRIPT_DIR" > /dev/null || exit
    RELEASE_ZIP="$WORK_DIR/piCore64-$PICORE_SUBVERSION.zip"

    if [ -f "$RELEASE_ZIP" ]; then
        echo " * PiCore64 release zip already exists. Unzipping..."
        unzip -o "$RELEASE_ZIP" -d $WORK_DIR
        check_release
    else
        if validate_url $PICORE_RELEASE_URL; then
            echo -ne " * PiCore64 $PICORE_SUBVERSION" "($PICORE_RELEASE_URL)"    
            echo ""
            read -n1 -r -p " * Press any key to download..." key
            echo ""
            wget $WGET_OPTS "$PICORE_RELEASE_URL" -P "$WORK_DIR" &&
            echo "- unzipping"
            unzip -o "$WORK_DIR/piCore64-$PICORE_SUBVERSION.zip" -d $WORK_DIR
            check_release
        else
            echo " * ERROR: url not available"
            kill -s TERM $TOP_PID
        fi
    fi
    echo ""
}

function check_release(){
    echo "================================================================================" 
    echo " * Checking release"
    echo "================================================================================" 
    if [ -f "$WORK_DIR/piCore64-$PICORE_SUBVERSION.zip" ]; then 
        cd "$WORK_DIR" || exit
        if md5sum --status -c "piCore64-$PICORE_SUBVERSION.img.md5.txt"; then
            echo "  - Release available: $WORK_DIR/piCore64-$PICORE_SUBVERSION.img"
        else
            echo "  - Checksum FAILED: piCore64-$PICORE_SUBVERSION.img.md5.txt"
            download_release_maybe
        fi
    else 
        download_release_maybe
    fi
    cd "$SCRIPT_DIR" > /dev/null || exit
    echo ""
}

function make_image(){
    echo "================================================================================" 
    echo " * Creating Custom PiCore64 Image"
    echo "================================================================================" 
    echo "  - generating empty image (be patient)"
    cd "$SCRIPT_DIR" > /dev/null || exit
    sudo touch $WORK_DIR/piCore64-$PICORE_SUBVERSION.$PICORE_APP.img
    sudo dd bs=$IMG_BLOCKSIZE count=$IMG_BLOCKS if=/dev/zero of=$WORK_DIR/piCore64-$PICORE_SUBVERSION.$PICORE_APP.img
    echo ""    
    echo "  - cloning into $PICORE_APP image (be patient)"
    SRC="$(sudo losetup -f --show $WORK_DIR/piCore64-$PICORE_SUBVERSION.img)"
    DEST="$(sudo losetup -f --show $WORK_DIR/piCore64-$PICORE_SUBVERSION.$PICORE_APP.img)"
    sudo dd if=$SRC of=$DEST

    echo "  - init $PICORE_APP image loop device"
    sudo losetup -d $SRC
    sudo losetup -d $DEST
    
    echo "  - setting up $PICORE_APP image partitions"
    rm -rf "$WORK_DIR/piCore64-$PICORE_SUBVERSION.$PICORE_APP.img1"
    rm -rf "$WORK_DIR/piCore64-$PICORE_SUBVERSION.$PICORE_APP.img2"
    tmp=$(sudo kpartx -l "$WORK_DIR/piCore64-$PICORE_SUBVERSION.$PICORE_APP.img" | awk '{ print $1 }' )
    IFS=$'\n' read -rd '' -a parts <<<"$tmp"

    echo "  - trying kpartx (adding...)"
    sudo kpartx -a "$WORK_DIR/piCore64-$PICORE_SUBVERSION.$PICORE_APP.img"
    sleep 3
    ln -s /dev/mapper/"${parts[0]}" "$WORK_DIR/piCore64-$PICORE_SUBVERSION.$PICORE_APP.img1" &> /dev/null
    ln -s /dev/mapper/"${parts[1]}" "$WORK_DIR/piCore64-$PICORE_SUBVERSION.$PICORE_APP.img2" &> /dev/null

    echo "  - resizing $PICORE_APP image partition"
    tmp=$(sudo parted -m -s "$WORK_DIR/piCore64-$PICORE_SUBVERSION.$PICORE_APP.img" unit s print | awk -F: '{print $2}')
    IFS=$'\n' read -rd '' -a size <<<"$tmp"
    start=${size[2]::-1}
    end=$((${size[0]::-1}-1))

    echo "  - trying parted"
    sudo parted -s "$WORK_DIR/piCore64-$PICORE_SUBVERSION.$PICORE_APP.img" unit s rm 2
    sudo parted -s "$WORK_DIR/piCore64-$PICORE_SUBVERSION.$PICORE_APP.img" unit s mkpart primary "$start" $end

    echo "  - trying kpartx (cleaning up...)"
    sudo kpartx -d "$WORK_DIR/piCore64-$PICORE_SUBVERSION.$PICORE_APP.img" &> /dev/null
    sleep 3

    echo "  - trying kpartx (adding...)"
    sudo kpartx -a "$WORK_DIR/piCore64-$PICORE_SUBVERSION.$PICORE_APP.img"
    sleep 3
    ln -s /dev/mapper/"${parts[0]}" "$WORK_DIR/piCore64-$PICORE_SUBVERSION.$PICORE_APP.img1" &> /dev/null
    ln -s /dev/mapper/"${parts[1]}" "$WORK_DIR/piCore64-$PICORE_SUBVERSION.$PICORE_APP.img2" &> /dev/null

    echo ""
    echo "  - checking filesystem ($PICORE_APP image)"
    sudo e2fsck -f "$WORK_DIR/piCore64-$PICORE_SUBVERSION.$PICORE_APP.img2"

    echo ""
    echo "  - resizing filesystem ($PICORE_APP image)"
    sudo resize2fs "$WORK_DIR/piCore64-$PICORE_SUBVERSION.$PICORE_APP.img2"

    echo ""
    echo "  - mounting partition n. 1 ($PICORE_APP image)"
    if mount | grep "$MNT1_DIR" > /dev/null; then
        echo "Partition 1 already mounted. Unmounting..."
        sudo umount $MNT1_DIR
    fi
    sudo mount "$WORK_DIR/piCore64-$PICORE_SUBVERSION.$PICORE_APP.img1" $MNT1_DIR
    if [ $? -ne 0 ]; then
        echo "Error mounting partition 1. Exiting..."
        cleanup
        exit 1
    fi
    echo "  - mounting partition n. 2 ($PICORE_APP image)"
    if mount | grep "$MNT2_DIR" > /dev/null; then
        echo "Partition 2 already mounted. Unmounting..."
        sudo umount $MNT2_DIR
    fi
    sudo mount "$WORK_DIR/piCore64-$PICORE_SUBVERSION.$PICORE_APP.img2" $MNT2_DIR
    if [ $? -ne 0 ]; then
        echo "Error mounting partition 2. Exiting..."
        cleanup
        exit 1
    fi
}


function cleanup(){
    echo "================================================================================" 
    echo " * Cleaning up"
    echo "================================================================================"
    # Umount partition 1 and delete image
    if [ -d "$MNT1_DIR" ]; then 
        sudo umount "$MNT1_DIR" &> /dev/null
        rm -rf "$MNT1_DIR"
    fi
    # Umount partition 2 and delete image
    if [ -d "$MNT2_DIR" ]; then 
        sudo umount "$MNT2_DIR" &> /dev/null
        rm -rf "$MNT2_DIR"
    fi
    sudo kpartx -d "$WORK_DIR/piCore64-$PICORE_SUBVERSION.$PICORE_APP.img" &> /dev/null
    [ -L "$WORK_DIR/piCore64-$PICORE_SUBVERSION.$PICORE_APP.img1" ] && rm "$WORK_DIR/piCore64-$PICORE_SUBVERSION.$PICORE_APP.img1"
    [ -L "$WORK_DIR/piCore64-$PICORE_SUBVERSION.$PICORE_APP.img2" ] && rm "$WORK_DIR/piCore64-$PICORE_SUBVERSION.$PICORE_APP.img2"
    [ -d "$PICORE_FILESYSTEM_DIR" ] && sudo rm -rf "$PICORE_FILESYSTEM_DIR" 
    #[ -d "$MYDATA_DIR" ] && sudo rm -rf "$MYDATA_DIR"
    #[ -e "$WORK_DIR/piCore64-$PICORE_SUBVERSION.zip" ] && rm "$WORK_DIR/piCore64-$PICORE_SUBVERSION.zip"
    echo "  - OK"    
    echo ""
}

function test_package_urls(){
    echo "================================================================================" 
    echo " * Checking package URLs"
    echo "================================================================================" 
    for i in "${PICORE_PACKAGES[@]}"
    do
        URL="$PICORE_PACKAGES_URL/$i.$PICORE_PACKAGE_EXTENSION"
        echo -ne "  - $i" "($URL)"
        if validate_url "$URL"; then 
            echo " OK";
        else 
            echo " ERROR: url not available"; 
            cleanup
            kill -s TERM $TOP_PID
        fi
    done
}

function get_packages(){
    echo "================================================================================" 
    echo " * Downloading packages"
    echo "================================================================================" 
    for i in "${PICORE_PACKAGES[@]}"
    do
        URL="$PICORE_PACKAGES_URL/$i.$PICORE_PACKAGE_EXTENSION"
        echo "  - $i" "($URL)"
        if [ -f "$PACKAGES_DIR/$PICORE_LOCAL_PACKAGE_PATH/$i.tcz" ]; then
            echo " * Package available: $i"
        else
            sudo wget $WGET_OPTS "$URL" -P "$PACKAGES_DIR/$PICORE_LOCAL_PACKAGE_PATH/"
            sudo wget $WGET_OPTS "$URL.md5.txt" -P "$PACKAGES_DIR/$PICORE_LOCAL_PACKAGE_PATH/"
        fi
    done
    echo ""
    echo "  - Copying packages"
    sudo rsync -avz $PACKAGES_DIR/$PICORE_LOCAL_PACKAGE_PATH/* $MNT2_DIR/$PICORE_LOCAL_PACKAGE_PATH/
    echo "" 
}

function make_onboot_list(){
    echo "================================================================================" 
    echo " * Adding packages to onboot.lst"
    echo "================================================================================" 
    sudo sh -c "> $MNT2_DIR/tce/onboot.lst"
    for i in "${PICORE_PACKAGES[@]}"
    do
    	sudo sh -c "echo $i.tcz >> $MNT2_DIR/tce/onboot.lst"
    done
    
    # show onboot.lst
    sudo cat "$MNT2_DIR/tce/onboot.lst"
    echo " - OK"
    echo ""
}

function modify_boot_config(){
    echo "================================================================================" 
    echo " * Modifying config.txt"
    echo "================================================================================"

    # Ensure the config.txt file exists
    if [ -f "$BOOT_CONFIG" ]; then
       
        # Backup current config.txt
        sudo cp "$BOOT_CONFIG" "${BOOT_CONFIG}.bak"

   # Add "disable_splash=1" under [all]
sudo sed -i '/^\[ALL\]/a \
# Disable splash screen\ndisable_splash=1' "$BOOT_CONFIG"

# Uncomment dtparam=i2c_arm=on
sudo sed -i 's/^#\(dtparam=i2c_arm=on\)$/\1/' "$BOOT_CONFIG"

# Change dtparam=audio=on to dtparam=audio=off and add other parameters
sudo sed -i 's/^dtparam=audio=on$/dtparam=audio=off\
# General Device Tree Parameters\
dtparam=i2c_vc=on\
dtparam=spi=off/' "$BOOT_CONFIG"

# Add multiple dtoverlay parameters under [all]
sudo sed -i '/^\[all\]/a \
# Disable both WiFi and Bluetooth (no hardware)\
dtoverlay=disable-wifi\
dtoverlay=disable-bt\n\
# Set PCI DMA to 32bit\
dtoverlay=pcie-32bit-dma\n\
# Enable RTC\
dtoverlay=i2c-rtc,pcf85063a,i2c_csi_dsi\n\
# Enable fan controller set parameters\
dtoverlay=i2c-fan,emc2301,minpwm=0,maxpwm=255,midtemp=55000,maxtemp=75000' "$BOOT_CONFIG"


        echo "  - config.txt has been updated."

    fi

    echo ""
    echo "  - writing ssh & firsttime flags"
    echo ""
    sudo touch "$MNT1_DIR/ssh"
    sudo touch "$MNT1_DIR/firsttime"
    echo ""
    echo " - OK"
}

function make_mydata(){
    echo "================================================================================" 
    echo " * Adjusting mydata.tgz"
    echo "================================================================================" 
    echo "  - Unpacking mydata.tgz"
    [ -d "$MNT2_DIR/$PICORE_LOCAL_MYDATA" ] || sudo mkdir "$MNT2_DIR/$PICORE_LOCAL_MYDATA"
    sudo tar zxvf "$MNT2_DIR/$PICORE_LOCAL_MYDATA.tgz" -C "$MNT2_DIR/$PICORE_LOCAL_MYDATA"

    echo ""
   

    manage_scripts_and_directories

    echo ""
    echo "  - finalising"
    cd "$MNT2_DIR/$PICORE_LOCAL_MYDATA" || exit
    sudo tar -zcf ../mydata.tgz .
    cd "$SCRIPT_DIR" &> /dev/null || exit
    echo "  - deleting mydata"
    sudo rm -rf "$MNT2_DIR/$PICORE_LOCAL_MYDATA"
    echo ""
    echo " - OK"
}

function extract_filesystem(){
    echo "================================================================================" 
    echo " * Unpacking filesystem"
    echo "================================================================================" 
    echo "  - fixing permissions"
    sudo sh -c "chmod a+rwx $PICORE_FILESYSTEM_DIR"
    echo "  - checking if the gz file exists"
    if [ -f "${MNT1_DIR}/${PICORE_ROOTFS}.gz" ]; then
        echo "  - ${MNT1_DIR}/${PICORE_ROOTFS}.gz found, proceeding with extraction"
        sudo sh -c "zcat ${MNT1_DIR}/${PICORE_ROOTFS}.gz | (cd $PICORE_FILESYSTEM_DIR && sudo cpio -i -H newc -d)"
    else
        echo "Error: ${MNT1_DIR}/${PICORE_ROOTFS}.gz not found. Exiting..."
        cleanup
        exit 1
    fi
}

function rebuild_filesystem(){
    echo "================================================================================" 
    echo " * Rebuilding filesystem"
    echo "================================================================================" 
    sudo sh -c "(cd $PICORE_FILESYSTEM_DIR && find | cpio -o -H newc) | gzip -2 > ${MNT1_DIR}/${PICORE_SUBVERSION}.gz"
    echo ""
    echo " * $PICORE_APP image: $WORK_DIR/piCore64-$PICORE_SUBVERSION.$PICORE_APP.img" 
    echo ""
    echo " - OK"
}

check_dependencies
check_script_existence
cleanup
prepare_dirs
check_release
make_image
test_package_urls
get_packages
make_onboot_list
make_mydata
modify_boot_config
extract_filesystem
rebuild_filesystem
cleanup