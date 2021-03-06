// MIT License
//
// Copyright (c) 2018 the io-core authors
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"flag"
	"fmt"
	"github.com/io-core/Attest/s2r"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const atestline = "----Attest-0.1.0------------------------------------------------------------------------"
const seperator = "----------------------------------------------------------------------------------------"
const formatc = `
--,--,ada
//,//,actionscript
--,--,applescript
# , #,assembly
# , #,bash
/*,*/,c
# , #,c#
//,//,c++
; , ;,clojure
# , #,coffeescript
/*,*/,css
//,//,delphi
% ,  ,erlang
! , !,f90
C , C,FORTRAN
//,//,go
--,--,haskell
  ,  ,haskellb
<!--, -->,html
! ,  ,ios
//,//,java
//,//,javascript
--,--,lua
% , %,matlab
# , #,shell
(*,*),modula2
(*,*),oberon
/*,*/,objectivec
(*,*),ocaml
(*,*),pascal
# , #,perl
//,//,php
# , #,powershell
# , #,python
# , #,ruby
--,--,sql
//,//,scala
//,//,swift
{ , },tpascal
' , ',vb
<!--, -->,xml
`



func getKeys(pkfn, bkfn string) (*rsa.PrivateKey, string) {

	pk, _ := ioutil.ReadFile(pkfn)
	bk, _ := ioutil.ReadFile(bkfn)
	bks := strings.TrimSpace(string(bk))
	privPem, _ := pem.Decode(pk)
	privPemBytes := privPem.Bytes
	parsedKey, _ := x509.ParsePKCS1PrivateKey(privPemBytes)
	return parsedKey, bks
}

func sign(contents []byte, asserts, cl, cr, pkeyf, bkeyf string) {

	al := strings.Split(asserts, ",")
	trail := "\n"
	for _, v := range al {
		trail = trail + v + "\n"
	}

	now := fmt.Sprint(time.Now().Format("2006-01-02 15:04:05"))
	trail = trail + now + "\n"
	message := append(contents, trail...)
	hashed := sha256.Sum256(message)
	parsedKey, bks := getKeys(pkeyf, bkeyf)

	signature, err := rsa.SignPKCS1v15(rand.Reader, parsedKey, crypto.SHA256, hashed[:])
	if err != nil {
		fmt.Println(err)
	}

	spaces := "                                                                                                    "
	
	encoded := base64.StdEncoding.EncodeToString(signature)
	fmt.Println("\n" + cl + atestline + cr)
	for _, v := range al {
		fmt.Println(cl, v, spaces[:85-len(v)], cr)
	}
	fmt.Println(cl, now, spaces[:85-len(now)], cr)
	fmt.Println(cl + seperator + cr)
	emit(encoded, spaces, cl, cr)
	fmt.Println(cl + seperator + cr)
	emit(bks, spaces, cl, cr)
	fmt.Println(cl + seperator + cr)

}

func emit(s, spaces, cl, cr string) {

	for x := 0; x < len(s); x = x + 86 {
		mo := min(x+86, len(s))
		trail := ""
		if mo != x+86 {
			trail = spaces[:(x+86)-len(s)-1]
			fmt.Println(cl, s[x:mo], trail, cr)
		} else {
			fmt.Println(cl, s[x:mo], cr)
		}

	}

}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func shave(s, d string) string {
	r := ""
	x := strings.Split(s, "\n")
	for i, v := range x {
		r = r + strings.TrimSpace(v[3:len(v)-3])
		if i < len(x) {
			r = r + d
		}
	}
	return r
}

func findSig(contents []byte) (o int, al, hl, bl string) {

	s := strings.Split(string(contents), atestline)
	sig := ""
	cl := ""
	cr := ""

	if len(s) > 1 {
		for i := 0; i < len(s)-1; i++ {
			if len(s[i]) > 1 && len(s[i+1]) > 1 {
				if (s[i][len(s[i])-3:len(s[i])-2] == "\n" && s[i+1][2:3] == "\n") {
					for j := 0; j <= i; j++ {
						o = o + len(s[j]) + len(atestline)
					}
					o = o - len(atestline) - 3

					sig = s[i+1][2:]
					cl = s[i][len(s[i])-2:]
					cr = s[i+1][0:2]
				}
			}
		}
	}
	sep := "\n" + cl + seperator + cr + "\n"
	sect := strings.Split(sig, sep)
	if len(sect)<3 {
		fmt.Println("Signature not found!")
		os.Exit(1)
	}else{
		al = shave(sect[0][1:], "\n")
		hl = shave(sect[1], "")
		bl = shave(sect[2], "")
	}
	return o, al, hl, bl
}

func check(contents []byte,tkfn string) {

	o, al, hl, bl := findSig(contents)
	message := append(contents[:o], "\n"+al...)
	hashed := sha256.Sum256(message)
	signature, err := base64.StdEncoding.DecodeString(hl)
	if err != nil {
		fmt.Println(err)
	} else {

		pubKeyString := s2r.Translate(bl)

		block, berr := pem.Decode([]byte(pubKeyString))

		if berr == nil {
			fmt.Println(berr)
			panic("failed to parse PEM block containing the public key")
		}

		rpk, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			panic("failed to parse DER encoded public key: " + err.Error())
		}

		rsaPubKey := rpk.(*rsa.PublicKey)
		err = rsa.VerifyPKCS1v15(rsaPubKey, crypto.SHA256, hashed[:], signature)

		if err != nil {
			fmt.Println("verify error:", err)
			os.Exit(1)
		} else {
			tk, _ := ioutil.ReadFile(tkfn)
			tkeys:=strings.Split(string(tk),"\n")
			found := false
			for _,v := range tkeys {
				if v == bl {
					found = true
				}
			}
			if found {
				fmt.Println("verify success!")
				os.Exit(0)
			}else{
                                fmt.Println("public key of signature not found in trusted_devs file")
                                os.Exit(2)
			}
		}
	}

}


func findFormat(s string)(l,r string){
	if s == "csv" {
		l=" ,"
		r=", "
	}else{	
        	for _,v := range strings.Split( formatc, "\n"){
                	e:= strings.Split( v, ",")
                	if len(e)==3 {
                	        if e[2] == s {
					l=e[0]
					r=e[1]
				}
                	}
        	}
	}
	return l,r
}


func main() {
	
	inFilePtr := flag.String("i", "-", "input file")
	aMessagePtr := flag.String("a", "signed", "attest message")
	formatPtr := flag.String("f", "oberon", "attest comment style")
	pkeyPtr := flag.String("p", os.Getenv("HOME")+"/.ssh/id_rsa", "path to rsa private key file")
	bkeyPtr := flag.String("b", os.Getenv("HOME")+"/.ssh/id_rsa.pub", "path to rsa public key file")
        tkeysPtr := flag.String("t", os.Getenv("HOME")+"/.ssh/trusted_devs", "path to trusted_devs file")
	checkPtr := flag.Bool("c", false, "check instead of sign")
        rkeyPtr := flag.Bool("k", false, "retrieve public key from input file")

	flag.Parse()

	

	iam := filepath.Base(os.Args[0])
	if iam == "acheck" {
		f := true
		checkPtr = &f
	}

	tail:= flag.Args()
	var contents []byte
	if len(tail)>0 {
	  contents, _ = ioutil.ReadFile(tail[0])
	}else{
	  contents, _ = ioutil.ReadFile(*inFilePtr)
	}

	if *rkeyPtr {
	        _, _, _, bl := findSig(contents)
	        fmt.Println(bl)
        } else if *checkPtr {
                check(contents,*tkeysPtr)
	} else {
		l,r := findFormat(*formatPtr)
		sign(contents, *aMessagePtr, l, r, *pkeyPtr, *bkeyPtr)
	}
}

//----Attest-0.1.0------------------------------------------------------------------------//
// signed                                                                                 //
// 2019-03-13 09:57:24                                                                    //
//----------------------------------------------------------------------------------------//
// BkVpb/Yrtar+2ezFSTxH89abN3+bVNdyzW0g2OAeV6nEvKgJjYo/m0fI7E2cYuGDLPOmsG/UREJQiw8gpf3Zaq //
// BMkaYnkNhVnBdGgmmUgDKgPn59daXyDnttL4v/7t1SOLNvlxUFcUxw9A4aPRvKcjAokna8+tu34UfAhqGoIf20 //
// bsLaXOGr0PZw6Z2zNnNMWSe91+A45DyX56yI4/9MwOl9L6UnMx7li6IMLMh8WEDq+nKIes2CZCHFb9GFhWLqXn //
// YiQF+ClVX0/ybBbDSRr56jrKPlKC48PTsmlPkjGbgbTrgMOp6BmzygYzGyrPJ3g41AXew9TYVP8+bziFctgA== //
//----------------------------------------------------------------------------------------//
// ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDsrtAUhLbs/ELXgH3OJs0SKh7tSQE/gkPavHv4//tsLucTAN //
// C4mEjbjxKtFlZjji89GGvatnGu3DvAAz60VNEGBccezdn4rkcNpceKQe2KE2Kb13KM6VmrNl4Gj3+C278u0yKx //
// l07WpQCYJ1x6WU8Tnrs5oRSGvHzJVvkxbH7YfymnoXbDg2j8cWYX+zNR/aYvcX+6isZmqRDg+qZ1CK45UL0sO9 //
// GcSFyey3fGigzWGvBx9JujvsxL6aqX7yY+WtCbApeGLN4HYtrn4ueuKAQND5EYo0SEI2m+STt5eCdDBLFhG0jD //
// 5MO6T7o//Mg8qDeuiY5wpbcQdpVWmdWQQxMT chuck@kuracali.com                                //
//----------------------------------------------------------------------------------------//
