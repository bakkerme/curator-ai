import type React from "react"

interface StarscapeBackgroundProps {
  className?: string
  opacity?: number
  transform?: string
  transformOrigin?: string
  transition?: string
}

export const StarscapeBackground: React.FC<StarscapeBackgroundProps> = ({ 
  className = "starscape", 
  opacity = 0.6,
  transform,
  transformOrigin = "0 0",
  transition
}) => {
  return (
    <div 
      className={`absolute inset-0 ${className}`}
      style={{
        transform,
        transformOrigin,
        transition
      }}
    >
      {/* Starscape Background - Infinitely Tileable */}
      <div
        className="absolute inset-0 w-full h-full"
        style={{
          opacity,
          background: `
            radial-gradient(2px 2px at 20px 30px, #fff, transparent),
            radial-gradient(2px 2px at 40px 70px, rgba(255,255,255,0.8), transparent),
            radial-gradient(1px 1px at 90px 40px, #fff, transparent),
            radial-gradient(1px 1px at 130px 80px, rgba(255,255,255,0.6), transparent),
            radial-gradient(2px 2px at 160px 30px, #fff, transparent),
            radial-gradient(1px 1px at 200px 90px, rgba(255,255,255,0.8), transparent),
            radial-gradient(1px 1px at 240px 50px, #fff, transparent),
            radial-gradient(2px 2px at 280px 120px, rgba(255,255,255,0.7), transparent),
            radial-gradient(1px 1px at 320px 40px, #fff, transparent),
            radial-gradient(1px 1px at 360px 100px, rgba(255,255,255,0.6), transparent),
            radial-gradient(2px 2px at 400px 20px, #fff, transparent),
            radial-gradient(1px 1px at 440px 80px, rgba(255,255,255,0.8), transparent),
            radial-gradient(1px 1px at 480px 60px, #fff, transparent),
            radial-gradient(2px 2px at 520px 110px, rgba(255,255,255,0.7), transparent),
            radial-gradient(1px 1px at 560px 30px, #fff, transparent),
            radial-gradient(1px 1px at 600px 90px, rgba(255,255,255,0.6), transparent),
            radial-gradient(2px 2px at 640px 50px, #fff, transparent),
            radial-gradient(1px 1px at 680px 120px, rgba(255,255,255,0.8), transparent),
            radial-gradient(1px 1px at 720px 40px, #fff, transparent),
            radial-gradient(2px 2px at 760px 100px, rgba(255,255,255,0.7), transparent),
            radial-gradient(1px 1px at 60px 150px, #fff, transparent),
            radial-gradient(2px 2px at 100px 200px, rgba(255,255,255,0.8), transparent),
            radial-gradient(1px 1px at 140px 180px, #fff, transparent),
            radial-gradient(1px 1px at 180px 220px, rgba(255,255,255,0.6), transparent),
            radial-gradient(2px 2px at 220px 160px, #fff, transparent),
            radial-gradient(1px 1px at 260px 240px, rgba(255,255,255,0.8), transparent),
            radial-gradient(1px 1px at 300px 190px, #fff, transparent),
            radial-gradient(2px 2px at 340px 250px, rgba(255,255,255,0.7), transparent),
            radial-gradient(1px 1px at 380px 170px, #fff, transparent),
            radial-gradient(1px 1px at 420px 230px, rgba(255,255,255,0.6), transparent),
            radial-gradient(2px 2px at 460px 200px, #fff, transparent),
            radial-gradient(1px 1px at 500px 260px, rgba(255,255,255,0.8), transparent),
            radial-gradient(1px 1px at 540px 180px, #fff, transparent),
            radial-gradient(2px 2px at 580px 240px, rgba(255,255,255,0.7), transparent),
            radial-gradient(1px 1px at 620px 210px, #fff, transparent),
            radial-gradient(1px 1px at 660px 270px, rgba(255,255,255,0.6), transparent),
            radial-gradient(2px 2px at 700px 190px, #fff, transparent),
            radial-gradient(1px 1px at 740px 250px, rgba(255,255,255,0.8), transparent)
          `,
          backgroundRepeat: "repeat",
          backgroundSize: "800px 300px",
        }}
      />

      {/* Dark Grid Background - Infinitely Tileable */}
      <div
        className="absolute inset-0 w-full h-full opacity-40"
        style={{
          backgroundImage: `
            linear-gradient(rgba(255,255,255,0.15) 1px, transparent 1px),
            linear-gradient(90deg, rgba(255,255,255,0.15) 1px, transparent 1px)
          `,
          backgroundSize: "40px 40px",
        }}
      />

      {/* Subtle Glow Effect */}
      <div className="absolute inset-0 w-full h-full bg-gradient-to-br from-yellow-500/5 via-transparent to-blue-500/5" />
    </div>
  )
}
