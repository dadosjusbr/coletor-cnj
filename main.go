// package main

// import (
//     "fmt"
//     "io/ioutil"
//     "os"
// )

// func main() {
//     dir := "output/"
//     files, _ := ioutil.ReadDir(dir)
//     var newestFile string
//     var newestTime int64 = 0
//     for _, f := range files {
//         fi, err := os.Stat(dir + f.Name())
//         if err != nil {
//             fmt.Println(err)
//         }
//         currTime := fi.ModTime().Unix()
//         if currTime > newestTime {
//             newestTime = currTime
//             newestFile = f.Name()
//         }
//     }
//     fmt.Println(newestFile)
// }
