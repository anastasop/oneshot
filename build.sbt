lazy val root = (project in file("."))
  .settings( 
    name := "first",
    version := "1.0-SNAPSHOT",

    scalaVersion := "2.11.11",

    libraryDependencies ++= Seq(
//      "com.typesafe.play" %% "play" % "2.5.18" 
    ),

    resolvers ++= Seq(
      "Typesafe repository" at "http://repo.typesafe.com/typesafe/releases/",
      "Sonatype releases" at "https://oss.sonatype.org/content/repositories/releases"
    )
  )

